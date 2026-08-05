package lua

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flanksource/is-healthy/pkg/health"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type TestStructure struct {
	Tests []IndividualTest `yaml:"tests"`
}

type IndividualTest struct {
	InputPath    string              `yaml:"inputPath"`
	HealthStatus health.HealthStatus `yaml:"healthStatus"`
}

func getObj(path string) *unstructured.Unstructured {
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	obj := make(map[string]interface{})
	err = yaml.Unmarshal(yamlBytes, &obj)
	if err != nil {
		panic(err)
	}

	return &unstructured.Unstructured{Object: obj}
}

func TestLuaHealthScript(t *testing.T) {
	err := filepath.Walk("../resource_customizations", func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.Contains(path, "health.lua") {
			return nil
		}
		dir := filepath.Dir(path)
		yamlBytes, err := os.ReadFile(dir + "/health_test.yaml")
		if err != nil {
			return err
		}
		var resourceTest TestStructure
		err = yaml.Unmarshal(yamlBytes, &resourceTest)
		if err != nil {
			return err
		}
		for i := range resourceTest.Tests {
			test := resourceTest.Tests[i]
			t.Run(test.InputPath, func(t *testing.T) {
				vm := VM{
					UseOpenLibs: true,
				}
				obj := getObj(filepath.Join(dir, test.InputPath))
				script, _, err := vm.GetHealthScript(obj)
				if err != nil {
					t.Error(err)
					return
				}
				result, err := vm.ExecuteHealthLua(obj, script)
				if err != nil {
					t.Error(err)
					return
				}
				expected := test.HealthStatus
				normalizeHealthStatus(&expected)
				assert.Equal(t, &expected, result)
			})
		}
		return nil
	})
	assert.Nil(t, err)
}
