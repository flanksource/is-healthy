package health

import (
	_ "embed"
	"strings"

	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:embed statusMap.yaml
var statusYaml []byte

var statusByKind map[string]StatusMap

func init() {
	statusByKind = make(map[string]StatusMap)
	if err := yaml.Unmarshal(statusYaml, &statusByKind); err != nil {
		panic(err.Error())
	}
}

type status struct {
	Status struct {
		Conditions []metav1.Condition
	}
}

type GenericStatus struct {
	Conditions []metav1.Condition
	Fields     map[string]interface{}
}

func (s GenericStatus) IsEqualInt(a, b string) bool {
	aInt, aOk := s.Int(a)
	bInt, bOk := s.Int(b)
	return aOk && bOk && aInt == bInt
}

func (s GenericStatus) Int(name string) (int32, bool) {
	value, ok := s.Fields[name]
	if !ok {
		return 0, false
	}

	switch v := value.(type) {
	case int32:
		return v, true
	case int64:
		return int32(v), true
	}
	return 0, false
}

func (s GenericStatus) FindCondition(name string) *metav1.Condition {
	if name == "" || name == NoCondition {
		return nil
	}
	// FindStatusCondition finds the conditionType in conditions.
	for i := range s.Conditions {
		if s.Conditions[i].Type == name {
			return &s.Conditions[i]
		}
	}
	return nil
}

func GetGenericStatus(obj *unstructured.Unstructured) GenericStatus {
	s := GenericStatus{}
	if objStatus, ok := obj.Object["status"].(map[string]any); ok {
		s.Fields = objStatus
	}

	holder := status{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &holder)
	if err != nil {
		return s
	}
	s.Conditions = holder.Status.Conditions
	return s
}

type OnCondition struct {
	// When 2 conditions are true, which one takes precedence from a status/message perspective
	Order int `yaml:"order,omitempty" json:"order,omitempty"`
	// If the condition matches, mark ready
	Ready bool `yaml:"ready"           json:"ready"`

	// If the condition matches, mark not ready
	NotReady bool `json:"notReady" yaml:"notReady,omitempty"`

	// If the condition is true, use the conditions message
	Message bool   `json:"message"          yaml:"message"`
	Health  Health `json:"health,omitempty" yaml:"health,omitempty"`
	// Health to set if the condition is false

	Status HealthStatusCode `json:"status,omitempty" yaml:"status,omitempty"`
}

func (mapped *OnCondition) Apply(health *HealthStatus, c *metav1.Condition) {
	if mapped.Ready {
		health.Ready = true
	}

	if mapped.NotReady {
		health.Ready = false
	}

	if mapped.Health != "" {
		if mapped.Order >= health.order || health.Health == HealthUnknown {
			// Only set if order is higher or health is unknown
			health.Health = mapped.Health
		}
	}

	if mapped.Status != "" {
		if health.Status == "" || mapped.Order >= health.order {
			health.Status = mapped.Status
			health.order = mapped.Order
		}
	} else if c.Reason != "" {
		if health.Status == "" || mapped.Order >= health.order {
			health.Status = HealthStatusCode(c.Reason)
			health.order = mapped.Order
		}
	}

	if mapped.Message && c.Message != "" {
		if health.Message == "" || mapped.Order >= health.order {
			health.Message = c.Message
		}
	}
}

type Condition struct {
	OnCondition `yaml:",inline" json:",inline"`

	OnFalse   *OnCondition `yaml:"onFalse,omitempty"   json:"onFalse,omitempty"`
	OnUnknown *OnCondition `yaml:"onUnknown,omitempty" json:"onUnknown,omitempty"`

	// Custom settings per reason
	Reasons map[string]OnCondition `yaml:"reasons,omitempty" json:"reasons,omitempty"`
}

func (mapped *Condition) Apply(health *HealthStatus, c *metav1.Condition) {
	if c.Status == metav1.ConditionTrue {
		mapped.OnCondition.Apply(health, c)
	} else if c.Status == metav1.ConditionFalse && mapped.OnFalse != nil {
		mapped.OnFalse.Apply(health, c)
	} else if c.Status == metav1.ConditionFalse && mapped.OnFalse == nil {
		if mapped.Health == HealthHealthy {
			// if this is a healthy condition and no specific onFalse handling, mark unhealthy
			health.Health = HealthUnhealthy
			if mapped.Message {
				health.Message = c.Message
			}
			if health.Status == "" && c.Reason != "" {
				health.Status = HealthStatusCode(c.Reason)
			}
		}
		if mapped.Ready {
			if health.Status == "" && c.Reason != "" {
				health.Status = HealthStatusCode(c.Reason)
			}
		}
		if mapped.Message {
			if health.Message == "" || mapped.Order >= health.order {
				health.Message = c.Message
			}
		}
	} else if c.Status == metav1.ConditionUnknown && mapped.OnUnknown != nil {
		mapped.OnUnknown.Apply(health, c)
	}

	if reason, ok := mapped.Reasons[c.Reason]; ok {
		reason.Apply(health, c)
	}
}

type StatusMap struct {
	Conditions          map[string]Condition `yaml:"conditions"          json:"conditions"`
	UnhealthyIsNotReady bool                 `yaml:"unhealthyIsNotReady" json:"unhealthyIsNotReady"`
}

const NoCondition = "none"

func GetDefaultHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	kind := obj.GetKind()
	group := strings.Split(obj.GetAPIVersion(), "/")[0]
	if strings.HasSuffix(group, "crossplane.io") || strings.HasSuffix(group, "upbound.io") {
		// For crossplane resources, we use a single status mapping under the dummy Kind "crossplane.io"
		// that is supposed to cater for all the crossplane kinds.
		kind = "crossplane.io"
	}

	if statusMap, ok := statusByKind[obj.GetAPIVersion()+"/"+obj.GetKind()]; ok {
		return GetHealthFromStatus(GetGenericStatus(obj), statusMap)
	} else if statusMap, ok := statusByKind[kind]; ok {
		return GetHealthFromStatus(GetGenericStatus(obj), statusMap)
	}

	return &HealthStatus{}, nil
}

func GetHealth(obj *unstructured.Unstructured, statusMap StatusMap) (*HealthStatus, error) {
	return GetHealthFromStatus(GetGenericStatus(obj), statusMap)
}

func GetHealthFromStatus(k GenericStatus, statusMap StatusMap) (*HealthStatus, error) {
	health := &HealthStatus{
		Health: HealthUnknown,
	}
	if len(statusMap.Conditions) == 0 {
		return health, nil
	}

	for _, condition := range k.Conditions {
		mappedCondition, ok := statusMap.Conditions[condition.Type]
		if ok {
			mappedCondition.Apply(health, &condition)
		}
	}

	if statusMap.UnhealthyIsNotReady && health.Health != HealthHealthy {
		health.Ready = false
	}

	return health, nil
}
