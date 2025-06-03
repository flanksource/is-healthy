package health

import (
	"fmt"
	"strings"
	"time"
)

var now = time.Now

const defaultAzureClientSecretExpiry = time.Hour * 24 * 30

var (
	azureClientSecretExpiry = defaultAzureClientSecretExpiry
)

func GetAzureHealth(configType string, obj map[string]any) HealthStatus {
	switch configType {
	case "Azure::AppRegistration::ClientSecret",
		"Azure::AppRegistration::Certificate":

		resourceType := strings.TrimPrefix(configType, "Azure::AppRegistration::")

		endDateTime := get(obj, "endDateTime")
		if endTime, err := time.Parse(time.RFC3339, endDateTime); err != nil {
			return HealthStatus{
				Health:  HealthUnknown,
				Message: "End date time is not set",
			}
		} else {
			if endTime.Before(now()) {
				return HealthStatus{
					Health:  HealthUnhealthy,
					Status:  "Expired",
					Message: fmt.Sprintf("%s has expired", resourceType),
				}
			}

			if now().Add(azureClientSecretExpiry).After(endTime) {
				return HealthStatus{
					Health:  HealthWarning,
					Status:  "Expiring",
					Message: fmt.Sprintf("%s is expiring in %s", resourceType, endTime.Sub(now())),
				}
			}
		}

		return HealthStatus{
			Health:  HealthHealthy,
			Status:  "Healthy",
			Message: fmt.Sprintf("%s is valid", resourceType),
		}

	default:
		return HealthStatus{
			Health: HealthUnknown,
		}
	}
}
