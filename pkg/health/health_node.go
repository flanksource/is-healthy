package health

import (
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getNodeHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var node v1.Node
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &node); err != nil {

		return nil, fmt.Errorf("failed to convert unstructured Node to typed: %v", err)
	}
	for _, cond := range node.Status.Conditions {
		if cond.Type == v1.NodeReady && cond.Status == v1.ConditionTrue {
			return &HealthStatus{
				Status: HealthStatusHealthy,
			}, nil
		}

		// All conditions apart from NodeReady should be false
		if cond.Status == v1.ConditionTrue {
			return &HealthStatus{
				Status:  HealthStatusDegraded,
				Message: fmt.Sprintf("%s: %s", cond.Type, cond.Message),
			}, nil
		}
	}
	return nil, errors.New("no conditions matched for node status")
}
