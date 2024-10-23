package main

import (
	"fmt"
	"io"
	"os"

	"github.com/flanksource/is-healthy/pkg/health"
	"github.com/flanksource/is-healthy/pkg/lua"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func main() {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	obj := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(stdin), &obj)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_health, err := health.GetResourceHealth(&unstructured.Unstructured{Object: obj}, lua.ResourceHealthOverrides{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("%s: %s\n", _health.Status, _health.Message)

	if health.IsWorse(health.HealthStatusHealthy, _health.Status) {
		os.Exit(1)
	}
}
