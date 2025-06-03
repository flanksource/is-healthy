package health

import (
	"fmt"
	"time"
)

const defaultAzureClientSecretExpiry = time.Hour * 24 * 30

var (
	azureClientSecretExpiry = defaultAzureClientSecretExpiry
)

func GetAzureHealth(configType string, obj map[string]any) HealthStatus {
	switch configType {
	case "Azure::ClientSecret":
		endDateTime := get(obj, "endDateTime")
		if endTime, err := time.Parse(time.RFC3339, endDateTime); err == nil {
			if endTime.Before(time.Now()) {
				return HealthStatus{
					Health:  HealthUnhealthy,
					Status:  "Expired",
					Message: "client secret has expired",
				}
			}

			if time.Now().Add(azureClientSecretExpiry).After(endTime) {
				return HealthStatus{
					Health:  HealthWarning,
					Status:  "Expiring",
					Message: fmt.Sprintf("client secret is expiring in %s", time.Until(endTime)),
				}
			}
		}

		return HealthStatus{
			Health:  HealthHealthy,
			Status:  "Healthy",
			Message: "Azure client secret is valid",
		}

	case "Azure::ClientCertificate":
		endDateTime := get(obj, "endDateTime")
		if endTime, err := time.Parse(time.RFC3339, endDateTime); err == nil {
			if endTime.Before(time.Now()) {
				return HealthStatus{
					Health:  HealthUnhealthy,
					Status:  "Expired",
					Message: "client certificate has expired",
				}
			}

			if time.Now().Add(azureClientSecretExpiry).After(endTime) {
				return HealthStatus{
					Health:  HealthWarning,
					Status:  "Expiring",
					Message: fmt.Sprintf("client certificate is expiring in %s", time.Until(endTime)),
				}
			}
		}
		return HealthStatus{
			Health:  HealthHealthy,
			Status:  "Healthy",
			Message: "Azure client certificate is valid",
		}

	default:
		return HealthStatus{
			Health: HealthUnknown,
		}
	}
}
