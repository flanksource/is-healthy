package lua

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	luajson "layeh.com/gopher-json"

	"github.com/flanksource/is-healthy/pkg/health"
	"github.com/flanksource/is-healthy/pkg/resource_customizations"
)

const (
	incorrectReturnType       = "expect %s output from Lua script, not %s"
	incorrectInnerType        = "expect %s inner type from Lua script, not %s"
	invalidHealthStatus       = "Lua returned an invalid health status"
	healthScriptFile          = "health.lua"
	actionScriptFile          = "action.lua"
	actionDiscoveryScriptFile = "discovery.lua"
)

type ResourceHealthOverrides map[string]ResourceOverride

func init() {
	health.DefaultOverrides = ResourceHealthOverrides{}
}

func (overrides ResourceHealthOverrides) GetResourceHealth(
	obj *unstructured.Unstructured,
) (*health.HealthStatus, error) {
	luaVM := VM{
		ResourceOverrides: overrides,
	}
	script, useOpenLibs, err := luaVM.GetHealthScript(obj)
	if err != nil {
		return nil, err
	}
	if script == "" {
		return nil, nil
	}
	// enable/disable the usage of lua standard library
	luaVM.UseOpenLibs = useOpenLibs
	result, err := luaVM.ExecuteHealthLua(obj, script)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// VM Defines a struct that implements the luaVM
type VM struct {
	ResourceOverrides map[string]ResourceOverride
	// UseOpenLibs flag to enable open libraries. Libraries are disabled by default while running, but enabled during testing to allow the use of print statements
	UseOpenLibs bool
}

func (vm VM) runLua(obj *unstructured.Unstructured, script string) (*lua.LState, error) {
	l := lua.NewState(lua.Options{
		SkipOpenLibs: !vm.UseOpenLibs,
	})
	defer l.Close()
	// Opens table library to allow access to functions to manipulate tables
	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		// load our 'safe' version of the OS library
		{lua.OsLibName, OpenSafeOs},
	} {
		if err := l.CallByParam(lua.P{
			Fn:      l.NewFunction(pair.f),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.n)); err != nil {
			panic(err)
		}
	}
	// preload our 'safe' version of the OS library. Allows the 'local os = require("os")' to work
	l.PreloadModule(lua.OsLibName, SafeOsLoader)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	l.SetContext(ctx)
	objectValue := decodeValue(l, obj.Object)
	l.SetGlobal("obj", objectValue)
	err := l.DoString(script)
	return l, err
}

// ExecuteHealthLua runs the lua script to generate the health status of a resource
func (vm VM) ExecuteHealthLua(obj *unstructured.Unstructured, script string) (*health.HealthStatus, error) {
	l, err := vm.runLua(obj, script)
	if err != nil {
		return nil, err
	}
	returnValue := l.Get(-1)
	if returnValue.Type() == lua.LTTable {

		jsonBytes, err := luajson.Encode(returnValue)
		if err != nil {
			return nil, err
		}
		healthStatus := &health.HealthStatus{}
		err = json.Unmarshal(jsonBytes, healthStatus)
		if err != nil {
			return nil, err
		}

		if healthStatus.Status != "" && healthStatus.Health == "" {
			switch healthStatus.Status {
			case health.HealthStatusUnknown:
				healthStatus.Health = health.HealthUnknown
				healthStatus.Status = ""
			case health.HealthStatusProgressing:
				healthStatus.Health = health.HealthUnknown
			case health.HealthStatusSuspended:
				healthStatus.Health = health.HealthUnknown
			case health.HealthStatusHealthy:
				healthStatus.Status = ""
				healthStatus.Health = health.HealthHealthy
				healthStatus.Ready = true
			case health.HealthStatusDegraded:
				healthStatus.Status = ""
				healthStatus.Health = health.HealthUnhealthy
				healthStatus.Ready = true
			case health.HealthStatusMissing:
				healthStatus.Ready = true
			}
		}
		healthStatus.Health = health.Health(strings.ToLower(string(healthStatus.Health)))

		return healthStatus, nil
	}
	return nil, fmt.Errorf(incorrectReturnType, "table", returnValue.Type().String())
}

// GetHealthScript attempts to read lua script from config and then filesystem for that resource
func (vm VM) GetHealthScript(obj *unstructured.Unstructured) (string, bool, error) {
	// first, search the gvk as is in the ResourceOverrides
	key := GetConfigMapKey(obj.GroupVersionKind())

	if script, ok := vm.ResourceOverrides[key]; ok && script.HealthLua != "" {
		return script.HealthLua, script.UseOpenLibs, nil
	}

	// if not found as is, perhaps it matches wildcard entries in the configmap
	wildcardKey := GetWildcardConfigMapKey(vm, obj.GroupVersionKind())

	if wildcardKey != "" {
		if wildcardScript, ok := vm.ResourceOverrides[wildcardKey]; ok && wildcardScript.HealthLua != "" {
			return wildcardScript.HealthLua, wildcardScript.UseOpenLibs, nil
		}
	}

	// if not found in the ResourceOverrides at all, search it as is in the built-in scripts
	// (as built-in scripts are files in folders, named after the GVK, currently there is no wildcard support for them)
	builtInScript, err := vm.getPredefinedLuaScripts(key, healthScriptFile)
	// standard libraries will be enabled for all built-in scripts
	return builtInScript, true, err
}

func (vm VM) ExecuteResourceAction(obj *unstructured.Unstructured, script string) ([]ImpactedResource, error) {
	l, err := vm.runLua(obj, script)
	if err != nil {
		return nil, err
	}
	returnValue := l.Get(-1)
	if returnValue.Type() == lua.LTTable {
		jsonBytes, err := luajson.Encode(returnValue)
		if err != nil {
			return nil, err
		}

		var impactedResources []ImpactedResource

		jsonString := bytes.NewBuffer(jsonBytes).String()
		if len(jsonString) < 2 {
			return nil, fmt.Errorf("lua output was not a valid json object or array")
		}
		// The output from Lua is either an object (old-style action output) or an array (new-style action output).
		// Check whether the string starts with an opening square bracket and ends with a closing square bracket,
		// avoiding programming by exception.
		if jsonString[0] == '[' && jsonString[len(jsonString)-1] == ']' {
			// The string represents a new-style action array output
			impactedResources, err = UnmarshalToImpactedResources(string(jsonBytes))
			if err != nil {
				return nil, err
			}
		} else {
			// The string represents an old-style action object output
			newObj, err := UnmarshalToUnstructured(string(jsonBytes))
			if err != nil {
				return nil, err
			}
			// Wrap the old-style action output with a single-member array.
			// The default definition of the old-style action is a "patch" one.
			impactedResources = append(impactedResources, ImpactedResource{newObj, PatchOperation})
		}

		for _, impactedResource := range impactedResources {
			// Cleaning the resource is only relevant to "patch"
			if impactedResource.K8SOperation == PatchOperation {
				impactedResource.UnstructuredObj.Object = cleanReturnedObj(
					impactedResource.UnstructuredObj.Object,
					obj.Object,
				)
			}
		}
		return impactedResources, nil
	}
	return nil, fmt.Errorf(incorrectReturnType, "table", returnValue.Type().String())
}

// UnmarshalToImpactedResources unmarshals an ImpactedResource array representation in JSON to ImpactedResource array
func UnmarshalToImpactedResources(resources string) ([]ImpactedResource, error) {
	if resources == "" || resources == "null" {
		return nil, nil
	}

	var impactedResources []ImpactedResource
	err := json.Unmarshal([]byte(resources), &impactedResources)
	if err != nil {
		return nil, err
	}
	return impactedResources, nil
}

// cleanReturnedObj Lua cannot distinguish an empty table as an array or map, and the library we are using choose to
// decoded an empty table into an empty array. This function prevents the lua scripts from unintentionally changing an
// empty struct into empty arrays
func cleanReturnedObj(newObj, obj map[string]interface{}) map[string]interface{} {
	mapToReturn := newObj
	for key := range obj {
		if newValueInterface, ok := newObj[key]; ok {
			oldValueInterface, ok := obj[key]
			if !ok {
				continue
			}
			switch newValue := newValueInterface.(type) {
			case map[string]interface{}:
				if oldValue, ok := oldValueInterface.(map[string]interface{}); ok {
					convertedMap := cleanReturnedObj(newValue, oldValue)
					mapToReturn[key] = convertedMap
				}

			case []interface{}:
				switch oldValue := oldValueInterface.(type) {
				case map[string]interface{}:
					if len(newValue) == 0 {
						mapToReturn[key] = oldValue
					}
				case []interface{}:
					newArray := cleanReturnedArray(newValue, oldValue)
					mapToReturn[key] = newArray
				}
			}
		}
	}
	return mapToReturn
}

// cleanReturnedArray allows Argo CD to recurse into nested arrays when checking for unintentional empty struct to
// empty array conversions.
func cleanReturnedArray(newObj, obj []interface{}) []interface{} {
	arrayToReturn := newObj
	for i := range newObj {
		switch newValue := newObj[i].(type) {
		case map[string]interface{}:
			if oldValue, ok := obj[i].(map[string]interface{}); ok {
				convertedMap := cleanReturnedObj(newValue, oldValue)
				arrayToReturn[i] = convertedMap
			}
		case []interface{}:
			if oldValue, ok := obj[i].([]interface{}); ok {
				convertedMap := cleanReturnedArray(newValue, oldValue)
				arrayToReturn[i] = convertedMap
			}
		}
	}
	return arrayToReturn
}

func (vm VM) ExecuteResourceActionDiscovery(obj *unstructured.Unstructured, script string) ([]ResourceAction, error) {
	l, err := vm.runLua(obj, script)
	if err != nil {
		return nil, err
	}
	returnValue := l.Get(-1)
	if returnValue.Type() == lua.LTTable {

		jsonBytes, err := luajson.Encode(returnValue)
		if err != nil {
			return nil, err
		}
		availableActions := make([]ResourceAction, 0)
		if noAvailableActions(jsonBytes) {
			return availableActions, nil
		}
		availableActionsMap := make(map[string]interface{})
		err = json.Unmarshal(jsonBytes, &availableActionsMap)
		if err != nil {
			return nil, err
		}
		for key := range availableActionsMap {
			value := availableActionsMap[key]
			resourceAction := ResourceAction{Name: key, Disabled: isActionDisabled(value)}
			if emptyResourceActionFromLua(value) {
				availableActions = append(availableActions, resourceAction)
				continue
			}
			resourceActionBytes, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(resourceActionBytes, &resourceAction)
			if err != nil {
				return nil, err
			}
			availableActions = append(availableActions, resourceAction)
		}
		return availableActions, err
	}

	return nil, fmt.Errorf(incorrectReturnType, "table", returnValue.Type().String())
}

// Actions are enabled by default
func isActionDisabled(actionsMap interface{}) bool {
	actions, ok := actionsMap.(map[string]interface{})
	if !ok {
		return false
	}
	for key, val := range actions {
		switch vv := val.(type) {
		case bool:
			if key == "disabled" {
				return vv
			}
		}
	}
	return false
}

func emptyResourceActionFromLua(i interface{}) bool {
	_, ok := i.([]interface{})
	return ok
}

func noAvailableActions(jsonBytes []byte) bool {
	// When the Lua script returns an empty table, it is decoded as a empty array.
	return string(jsonBytes) == "[]"
}

func (vm VM) GetResourceActionDiscovery(obj *unstructured.Unstructured) (string, error) {
	key := GetConfigMapKey(obj.GroupVersionKind())
	override, ok := vm.ResourceOverrides[key]
	if ok && override.Actions != "" {
		actions, err := override.GetActions()
		if err != nil {
			return "", err
		}
		return actions.ActionDiscoveryLua, nil
	}
	discoveryKey := fmt.Sprintf("%s/actions/", key)
	discoveryScript, err := vm.getPredefinedLuaScripts(discoveryKey, actionDiscoveryScriptFile)
	if err != nil {
		return "", err
	}
	return discoveryScript, nil
}

// GetResourceAction attempts to read lua script from config and then filesystem for that resource
func (vm VM) GetResourceAction(obj *unstructured.Unstructured, actionName string) (ResourceActionDefinition, error) {
	key := GetConfigMapKey(obj.GroupVersionKind())
	override, ok := vm.ResourceOverrides[key]
	if ok && override.Actions != "" {
		actions, err := override.GetActions()
		if err != nil {
			return ResourceActionDefinition{}, err
		}
		for _, action := range actions.Definitions {
			if action.Name == actionName {
				return action, nil
			}
		}
	}

	actionKey := fmt.Sprintf("%s/actions/%s", key, actionName)
	actionScript, err := vm.getPredefinedLuaScripts(actionKey, actionScriptFile)
	if err != nil {
		return ResourceActionDefinition{}, err
	}

	return ResourceActionDefinition{
		Name:      actionName,
		ActionLua: actionScript,
	}, nil
}

func GetConfigMapKey(gvk schema.GroupVersionKind) string {
	if gvk.Group == "" {
		return gvk.Kind
	}
	return fmt.Sprintf("%s/%s", gvk.Group, gvk.Kind)
}

func GetWildcardConfigMapKey(vm VM, gvk schema.GroupVersionKind) string {
	gvkKeyToMatch := GetConfigMapKey(gvk)

	for key := range vm.ResourceOverrides {
		if Match(key, gvkKeyToMatch) {
			return key
		}
	}
	return ""
}

func ListResourceTypes() []string {
	types := []string{}

	_ = fs.WalkDir(resource_customizations.Embedded, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && !strings.HasSuffix(d.Name(), "testdata") && !strings.Contains(path, "/actions") &&
			strings.Contains(path, "/") {
			types = append(types, path)
		}
		return nil
	})

	return types
}

func (vm VM) getPredefinedLuaScripts(objKey string, scriptFile string) (string, error) {
	data, err := resource_customizations.Embedded.ReadFile(filepath.Join(objKey, scriptFile))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Took logic from the link below and added the int, int32, and int64 types since the value would have type int64
// while actually running in the controller and it was not reproducible through testing.
// https://github.com/layeh/gopher-json/blob/97fed8db84274c421dbfffbb28ec859901556b97/json.go#L154
func decodeValue(L *lua.LState, value interface{}) lua.LValue {
	switch converted := value.(type) {
	case bool:
		return lua.LBool(converted)
	case float64:
		return lua.LNumber(converted)
	case string:
		return lua.LString(converted)
	case json.Number:
		return lua.LString(converted)
	case int:
		return lua.LNumber(converted)
	case int32:
		return lua.LNumber(converted)
	case int64:
		return lua.LNumber(converted)
	case []interface{}:
		arr := L.CreateTable(len(converted), 0)
		for _, item := range converted {
			arr.Append(decodeValue(L, item))
		}
		return arr
	case map[string]interface{}:
		tbl := L.CreateTable(0, len(converted))
		for key, item := range converted {
			tbl.RawSetH(lua.LString(key), decodeValue(L, item))
		}
		return tbl
	case nil:
		return lua.LNil
	}

	return lua.LNil
}
