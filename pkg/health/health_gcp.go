package health

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func GetGCPHealth(configType string, obj map[string]any) HealthStatus {
	switch configType {
	case "GCP::Compute::Disk":
		statusStr, found, err := unstructured.NestedString(obj, "status")
		if err != nil || !found {
			return HealthStatus{
				Health:  HealthUnknown,
				Message: fmt.Sprintf("GCP::Compute::Disk missing or invalid 'status' field: %v", err),
			}
		}

		// Extract size information using unstructured helper
		var message string
		if sizeStr, found, _ := unstructured.NestedString(obj, "sizeGb"); found {
			message = fmt.Sprintf("%s GB", sizeStr)
		} else {
			message = "No size information"
		}

		switch statusStr {
		case "READY":
			return HealthStatus{
				Health:  HealthHealthy,
				Status:  HealthStatusCode(lo.PascalCase(statusStr)),
				Message: message,
				Ready:   true,
			}
		case "CREATING":
			return HealthStatus{
				Health:  HealthUnknown,
				Status:  HealthStatusCode(lo.PascalCase(statusStr)),
				Message: message,
			}
		case "RESTORING":
			return HealthStatus{
				Health:  HealthWarning,
				Status:  HealthStatusCode(lo.PascalCase(statusStr)),
				Message: message,
			}
		case "FAILED":
			return HealthStatus{
				Health:  HealthUnhealthy,
				Status:  HealthStatusCode(lo.PascalCase(statusStr)),
				Message: message,
				Ready:   true,
			}
		case "DELETING":
			return HealthStatus{
				Health:  HealthWarning,
				Status:  HealthStatusCode(lo.PascalCase(statusStr)),
				Message: message,
			}
		default:
			return HealthStatus{
				Health:  HealthUnknown,
				Status:  HealthStatusCode(lo.PascalCase(statusStr)),
				Message: lo.CoalesceOrEmpty(message, "Unknown status"),
			}
		}

	case "GCP::Compute::InstanceGroupManager":
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
			return HealthStatus{
				Health:  HealthUnhealthy,
				Status:  HealthStatusDegraded,
				Message: message,
			}
		}

	case "GCP::Sqladmin::Instance":
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
		case "CREATING":
			return HealthStatus{
				Health:  HealthUnknown,
				Status:  HealthStatusCode(lo.PascalCase(stateStr)),
				Message: message,
			}
		case "SUSPENDED":
			return HealthStatus{
				Health:  HealthWarning,
				Status:  HealthStatusCode(lo.PascalCase(stateStr)),
				Message: message,
			}
		case "MAINTENANCE":
			return HealthStatus{
				Health:  HealthWarning,
				Status:  HealthStatusCode(lo.PascalCase(stateStr)),
				Message: message,
			}
		case "FAILED":
			return HealthStatus{
				Health:  HealthUnhealthy,
				Status:  HealthStatusCode(lo.PascalCase(stateStr)),
				Message: message,
			}
		default:
			return HealthStatus{
				Health:  HealthUnknown,
				Status:  HealthStatusCode(lo.PascalCase(stateStr)),
				Message: lo.CoalesceOrEmpty(message, "Unknown status"),
			}
		}
	}

	return HealthStatus{
		Health: HealthUnknown,
	}
}
