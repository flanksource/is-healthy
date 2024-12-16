package health

import (
	"regexp"
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var re = regexp.MustCompile(`(?:\((\d+\.?\d*)%\))|(\d+\.?\d*)%`)

func getCanaryHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	errorMsg, _, err := unstructured.NestedString(obj.Object, "status", "errorMessage")
	if err != nil {
		return nil, err
	}

	if errorMsg != "" {
		return &HealthStatus{
			Message: errorMsg,
			Health:  HealthUnhealthy,
		}, nil
	}

	message, _, _ := unstructured.NestedString(obj.Object, "status", "message")
	canaryStatus, _, _ := unstructured.NestedString(obj.Object, "status", "status")
	uptime1h, _, _ := unstructured.NestedString(obj.Object, "status", "uptime1h")

	output := HealthStatus{
		Message: message,
		Status:  HealthStatusCode(canaryStatus),
		Ready:   true,
	}

	switch canaryStatus {
	case "Passed":
		output.Health = HealthHealthy
		if uptime := parseCanaryUptime(uptime1h); uptime != nil && *uptime < float64(80) {
			output.Health = HealthWarning
		}
	case "Failed":
		output.Health = HealthUnhealthy
	case "Invalid":
		output.Health = HealthUnhealthy
		output.Ready = false // needs manual intervention
	}

	return &output, nil
}

func parseCanaryUptime(uptime string) *float64 {
	matches := re.FindStringSubmatch(uptime)
	var matched string
	if len(matches) > 0 {
		if matches[1] != "" {
			matched = matches[1]
		} else if matches[2] != "" {
			matched = matches[2]
		}
	}

	v, err := strconv.ParseFloat(matched, 64)
	if err != nil {
		return nil
	}

	return &v
}
