package health

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getNamespaceHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var node v1.Namespace
	if err := convertFromUnstructured(obj, &node); err != nil {
		return nil, err
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
