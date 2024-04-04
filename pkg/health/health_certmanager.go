package health

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	certificateExpiryWarningThreshold = 7 * 24 * time.Hour
)

func getCertificateRequestHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return nil, fmt.Errorf("failed to get conditions: %w", err)
	}

	if !found || len(conditions) == 0 {
		return &HealthStatus{
			Status:  HealthStatusPending,
			Message: "No conditions found",
		}, nil
	}

	var (
		lastCondition, _ = conditions[len(conditions)-1].(map[string]any)
		status, _        = lastCondition["status"].(string)
		conditionType, _ = lastCondition["type"].(string)
		message, _       = lastCondition["message"].(string)
	)
	if status == "True" && (conditionType == "Ready" || conditionType == "Approved") {
		return &HealthStatus{
			Status:  HealthStatusHealthy,
			Message: message,
		}, nil
	}

	return &HealthStatus{
		Status:  HealthStatusUnhealthy,
		Message: fmt.Sprintf("status=%s conditionType=%s message=%s", status, conditionType, message),
	}, nil
}

func getCertificateHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return nil, fmt.Errorf("failed to get conditions: %w", err)
	}

	notAfter, found, err := unstructured.NestedString(obj.Object, "status", "notAfter")
	if err != nil {
		return nil, fmt.Errorf("failed to get notAfter: %w", err)
	}

	if found {
		expiryTime, err := time.Parse(time.RFC3339, notAfter)
		if err != nil {
			return nil, fmt.Errorf("failed to parse notAfter: %w", err)
		}

		if expiryTime.Before(time.Now().Add(certificateExpiryWarningThreshold)) {
			return &HealthStatus{
				Status:  HealthStatusWarning,
				Message: fmt.Sprintf("Certificate expiring soon: %s", expiryTime.Format(time.RFC3339)),
			}, nil
		}
	}

	if !found || len(conditions) == 0 {
		return &HealthStatus{
			Status:  HealthStatusPending,
			Message: "No conditions found",
		}, nil
	}

	lastCondition := conditions[len(conditions)-1]
	cMap, _ := lastCondition.(map[string]any)
	cType, ok := cMap["type"].(string)
	if !ok || cType != "Ready" {
		return &HealthStatus{
			Status:  HealthStatusUnhealthy,
			Message: "Ready condition not found",
		}, nil
	}

	cStatus, ok := cMap["status"].(string)
	if !ok || cStatus != "True" {
		return &HealthStatus{
			Status:  HealthStatusUnhealthy,
			Message: "Ready condition not True",
		}, nil
	}

	message, _ := cMap["message"].(string)
	return &HealthStatus{
		Status:  HealthStatusHealthy,
		Message: message,
	}, nil
}
