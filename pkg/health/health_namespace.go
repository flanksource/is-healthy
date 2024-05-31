package health

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getNamespaceHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var node v1.Namespace
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &node); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured Node to typed: %v", err)
	}

	if node.Status.Phase == v1.NamespaceActive {
		return &HealthStatus{
			Ready:  true,
			Health: HealthUnknown,
			Status: HealthStatusHealthy,
		}, nil
	}

	return &HealthStatus{
		Health: HealthUnknown,
		Status: HealthStatusTerminating,
	}, nil
}
