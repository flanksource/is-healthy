package health

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/samber/lo"
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
		Message: lo.CoalesceOrEmpty(message, fmt.Sprintf("uptime: %s", uptime1h)),
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

func getNotificationHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	failedCount, _, err := unstructured.NestedInt64(obj.Object, "status", "failed")
	if err != nil {
		return nil, err
	}

	pendingCount, _, err := unstructured.NestedInt64(obj.Object, "status", "pending")
	if err != nil {
		return nil, err
	}

	errorMessage, _, err := unstructured.NestedString(obj.Object, "status", "error")
	if err != nil {
		return nil, err
	}

	sentCount, _, err := unstructured.NestedInt64(obj.Object, "status", "sent")
	if err != nil {
		return nil, err
	}

	var h Health = HealthUnknown
	if sentCount > 0 {
		h = HealthHealthy
		if failedCount > 0 || pendingCount > 0 {
			h = HealthWarning
		}
	} else {
		if pendingCount > 0 {
			h = HealthWarning
		}
		if failedCount > 0 {
			h = HealthUnhealthy
		}
	}

	status := &HealthStatus{Health: h}

	if errorMessage != "" {
		status.Message = errorMessage
	}

	return status, nil
}
