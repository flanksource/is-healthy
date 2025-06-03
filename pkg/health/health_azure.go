package health

import (
	"fmt"
	"time"
)

var now = time.Now

const defaultAzureClientSecretExpiry = time.Hour * 24 * 30

var (
	azureClientSecretExpiry = defaultAzureClientSecretExpiry
)

func GetAzureHealth(configType string, obj map[string]any) HealthStatus {
	switch configType {
	case "Azure::AppRegistration::ClientSecret":
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
					Message: "client secret has expired",
				}
			}

			if now().Add(azureClientSecretExpiry).After(endTime) {
				return HealthStatus{
					Health:  HealthWarning,
					Status:  "Expiring",
					Message: fmt.Sprintf("client secret is expiring in %s", endTime.Sub(now())),
				}
			}
		}

		return HealthStatus{
			Health:  HealthHealthy,
			Status:  "Healthy",
			Message: "Azure client secret is valid",
		}

	default:
		return HealthStatus{
			Health: HealthUnknown,
		}
	}
}
