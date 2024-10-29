package health

import (
	"errors"
	"fmt"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var defaultCertExpiryWarningPeriod = time.Hour * 24 * 2

func SetDefaultCertificateExpiryWarningPeriod(p time.Duration) {
	defaultCertExpiryWarningPeriod = p
}

func GetCertificateRequestHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, errors.New("certificate request doesn't have any conditions in the status")
	}

	for _, c := range conditions {
		cType := c.(map[string]any)["type"].(string)
		status := c.(map[string]any)["status"].(string)
		message := c.(map[string]any)["message"].(string)

		if cType == "Approved" && status == string(v1.ConditionTrue) {
			return &HealthStatus{
				Health:  HealthHealthy,
				Message: message,
				Status:  HealthStatusCode(cType),
				Ready:   true,
			}, nil
		}
	}

	status, err := GetDefaultHealth(obj)
	if err != nil {
		return status, err
	}

	return status, nil
}

func GetCertificateHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	if _notAfter, ok := obj.Object["status"].(map[string]any)["notAfter"]; ok {
		if notAfter := _notAfter.(string); ok {
			notAfterTime, err := time.Parse(time.RFC3339, notAfter)
			if err != nil {
				return nil, fmt.Errorf("failed to parse notAfter time(%s): %v", notAfter, err)
			}

			if notAfterTime.Before(time.Now()) {
				return &HealthStatus{
					Health:  HealthUnhealthy,
					Status:  "Expired",
					Message: "Certificate has expired",
					Ready:   true,
				}, nil
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

	status, err := GetDefaultHealth(obj)
	if err != nil {
		return status, err
	}

	return status, nil
}
