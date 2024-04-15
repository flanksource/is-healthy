package health

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getIngressHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	ingresses, found, err := unstructured.NestedSlice(obj.Object, "status", "loadBalancer", "ingress")
	if err != nil {
		return &HealthStatus{Status: HealthStatusError, Message: fmt.Sprintf("failed to ingress status: %v", err)}, nil
	} else if !found {
		return &HealthStatus{Health: HealthHealthy, Status: HealthStatusPending, Message: "ingress loadbalancer status not found"}, nil
	}

	health := HealthStatus{
		// Ready:  false, // not possible to decide this from the information available
		Health: HealthHealthy,
	}
	if len(ingresses) > 0 {
		health.Status = HealthStatusHealthy
		health.Health = HealthHealthy
		health.Ready = true

	} else {
		health.Status = HealthStatusPending
		health.Health = HealthUnknown
	}
	return &health, nil
}
