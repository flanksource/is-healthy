package health

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	defaultCertExpiryWarningPeriod = time.Hour * 24 * 2
)

func SetDefaultCertificateExpiryWarningPeriod(p time.Duration) {
	defaultCertExpiryWarningPeriod = p
}

func GetCertificateHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	if _renewalTime, ok := obj.Object["status"].(map[string]any)["renewalTime"]; ok {
		if renewalTimeString := _renewalTime.(string); ok {
			renewalTime, err := time.Parse(time.RFC3339, renewalTimeString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse renewal time (%s): %v", renewalTimeString, err)
			}

			if time.Since(renewalTime) > time.Minute*5 {
				return &HealthStatus{
					Health:  HealthWarning,
					Status:  HealthStatusWarning,
					Message: fmt.Sprintf("Certificate should have been renewed at %s", renewalTimeString),
					Ready:   true,
				}, nil
			}
		}
	}

	if _notAfter, ok := obj.Object["status"].(map[string]any)["notAfter"]; ok {
		if notAfter := _notAfter.(string); ok {
			notAfterTime, err := time.Parse(time.RFC3339, notAfter)
			if err != nil {
				return nil, fmt.Errorf("failed to parse notAfter time(%s): %v", notAfter, err)
			}

			if time.Until(notAfterTime) < defaultCertExpiryWarningPeriod {
				return &HealthStatus{
					Health:  HealthWarning,
					Status:  HealthStatusWarning,
					Message: fmt.Sprintf("Certificate is expiring soon (%s)", notAfter),
					Ready:   true,
				}, nil
			}
		}
	}

	status, err := GetDefaultHealth(obj)
	if err != nil {
		return status, err
	}

	return status, nil
}
