package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/flanksource/is-healthy/pkg/health"
	"github.com/flanksource/is-healthy/pkg/lua"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var jsonOut bool

func main() {
	if len(commit) > 8 {
		version = fmt.Sprintf("%v, commit %v, built at %v", version, commit[0:8], date)
	}

	root := &cobra.Command{
		Use: "is-healthy",
		RunE: func(cmd *cobra.Command, args []string) error {
			stdin, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}

			obj := make(map[string]interface{})
			err = yaml.Unmarshal([]byte(stdin), &obj)
			if err != nil {
				return err
			}

			_health, err := health.GetResourceHealth(&unstructured.Unstructured{Object: obj}, lua.ResourceHealthOverrides{})
			if err != nil {
				return err
			}

			if jsonOut {
				data, _ := json.MarshalIndent(_health, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("%s\n", *_health)
			}

			if _health.Health.IsWorseThan(health.HealthUnhealthy) {
				os.Exit(1)
			}
			if _health.Health.IsWorseThan(health.HealthWarning) {
				os.Exit(2)
			}
			return nil
		},
	}

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number of is-healthy",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "supported-types",
		Short: "Print a list of supported object types",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			for _, t := range lua.ListResourceTypes() {
				fmt.Println(t)
			}
			for _, t := range health.ListResourceTypes() {
				if strings.Contains(t, "/") {
					fmt.Println(t)
				} else {
					fmt.Println(t + "/*")
				}
			}
		},
	})

	root.SetUsageTemplate(root.UsageTemplate() + fmt.Sprintf("\nversion: %s\n ", version))

	root.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output in json format")
	if err := root.Execute(); err != nil {
		if jsonOut {
			data, _ := json.MarshalIndent(map[string]interface{}{"error": err.Error()}, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Println(err)
		}
		os.Exit(3)
	}
}
