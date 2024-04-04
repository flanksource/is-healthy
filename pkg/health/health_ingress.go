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
		return &HealthStatus{Status: HealthStatusPending, Message: "ingress loadbalancer status not found"}, nil
	}

	health := HealthStatus{}
	if len(ingresses) > 0 {
		health.Status = HealthStatusHealthy
	} else {
		health.Status = HealthStatusProgressing
	}
	return &health, nil
}
