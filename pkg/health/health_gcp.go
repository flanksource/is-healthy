package health

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

func GetGCPHealth(configType string, obj map[string]any) HealthStatus {
	switch configType {
	case "GCP::Compute::Disk":
		if status, ok := obj["status"]; ok {
			if statusStr, ok := status.(string); ok {
				sizeGb := ""
				if size, ok := obj["sizeGb"]; ok {
					if sizeStr, ok := size.(string); ok {
						sizeGb = fmt.Sprintf("%s GB", sizeStr)
					}
				}

				message := sizeGb

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
						Message: message,
					}
				}
			}
		}

	case "GCP::Compute::InstanceGroupManager":
		if status, ok := obj["status"]; ok {
			if statusMap, ok := status.(map[string]any); ok {
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
			}
		}

	case "GCP::Sqladmin::Instance":
		if state, ok := obj["state"]; ok {
			if stateStr, ok := state.(string); ok {
				var messageDetails []string
				if dbVersion, ok := obj["databaseVersion"]; ok {
					if dbVersionStr, ok := dbVersion.(string); ok {
						messageDetails = append(messageDetails, dbVersionStr)
					}
				}

				if settings, ok := obj["settings"]; ok {
					if settingsMap, ok := settings.(map[string]any); ok {
						if diskSize, ok := settingsMap["dataDiskSizeGb"]; ok {
							if diskSizeStr, ok := diskSize.(string); ok {
								messageDetails = append(messageDetails, fmt.Sprintf("%s GB", diskSizeStr))
							}
						}
					}
				}

				message := strings.Join(messageDetails, ", ")

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
						Message: message,
					}
				}
			}
		}
	}

	return HealthStatus{
		Health: HealthUnknown,
	}
}
