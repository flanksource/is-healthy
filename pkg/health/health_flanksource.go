package health

import (
	"regexp"
	"strconv"
	"strings"

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

func getScrapeConfigHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	errorCount, _, err := unstructured.NestedInt64(obj.Object, "status", "lastRun", "error")
	if err != nil {
		return nil, err
	}

	successCount, _, err := unstructured.NestedInt64(obj.Object, "status", "lastRun", "success")
	if err != nil {
		return nil, err
	}

	var h Health
	switch {
	case errorCount == 0 && successCount == 0:
		h = HealthUnknown
	case errorCount == 0 && successCount > 0:
		h = HealthHealthy
	case errorCount > 0 && successCount == 0:
		h = HealthUnhealthy
	case errorCount > 0 && successCount > 0:
		h = HealthWarning
	default:
		h = HealthUnknown
	}

	status := &HealthStatus{Health: h}

	if errorCount > 0 {
		errorMsgs, _, err := unstructured.NestedStringSlice(obj.Object, "status", "lastRun", "errors")
		if err != nil {
			return nil, err
		}

		if len(errorMsgs) > 0 {
			status.Message = strings.Join(errorMsgs, ",")
		}
	}

	return status, nil
}
