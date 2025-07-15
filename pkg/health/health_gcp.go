package health

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func GetGCPHealth(configType string, obj map[string]any) HealthStatus {
	switch configType {
	case "GCP::Disk":
		statusStr, found, err := unstructured.NestedString(obj, "status")
		if err != nil || !found {
			return HealthStatus{
				Health:  HealthUnknown,
				Message: fmt.Sprintf("GCP::Compute::Disk missing or invalid 'status' field: %v", err),
			}
		}

		var message string
		if sizeStr, found, _ := unstructured.NestedString(obj, "sizeGb"); found {
			message = fmt.Sprintf("%s GB", sizeStr)
		} else {
			message = "No size information"
		}

		healthStatus := GetHealthFromStatusName(statusStr, message)
		if healthStatus.Health != "" {
			return healthStatus
		}

		return HealthStatus{
			Health:  HealthUnknown,
			Message: message,
		}

	case "GCP::InstanceGroupManager":
		statusMap, found, err := unstructured.NestedMap(obj, "status")
		if err != nil || !found {
			return HealthStatus{
				Health:  HealthUnknown,
				Message: fmt.Sprintf("GCP::Compute::InstanceGroupManager missing or invalid 'status' field: %v", err),
			}
		}

		isStable := statusMap["isStable"] == true

		// Extract target size for meaningful message
		targetSize := 0
		if ts, ok := obj["targetSize"]; ok {
			switch v := ts.(type) {
			case float64:
				targetSize = int(v)
			case int:
				targetSize = v
			}
		}

		var message string
		if targetSize == 0 {
			message = "scaled to zero"
		} else {
			message = fmt.Sprintf("%d instances", targetSize)
		}

		if isStable {
			return HealthStatus{
				Health:  HealthHealthy,
				Status:  "Ready",
				Message: message,
				Ready:   true,
			}
		} else {
			return GetHealthFromStatusName("degraded", message)
		}

	case "GCP::SQLInstance":
		stateStr, found, err := unstructured.NestedString(obj, "state")
		if err != nil || !found {
			return HealthStatus{
				Health:  HealthUnknown,
				Message: fmt.Sprintf("GCP::Sqladmin::Instance missing or invalid 'state' field: %v", err),
			}
		}

		var messageDetails []string
		if dbVersionStr, found, _ := unstructured.NestedString(obj, "databaseVersion"); found {
			messageDetails = append(messageDetails, dbVersionStr)
		}

		if diskSizeStr, found, _ := unstructured.NestedString(obj, "settings", "dataDiskSizeGb"); found {
			messageDetails = append(messageDetails, fmt.Sprintf("%s GB", diskSizeStr))
		}

		message := lo.CoalesceOrEmpty(strings.Join(messageDetails, ", "), "No details available")
		switch stateStr {
		case "RUNNABLE":
			return HealthStatus{
				Health:  HealthHealthy,
				Status:  "Ready",
				Message: message,
				Ready:   true,
			}
		default:
			return GetHealthFromStatusName(stateStr, message)
		}
	}

	return HealthStatus{
		Health: HealthUnknown,
	}
}
