package health

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getNodeHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var node v1.Node
	if err := convertFromUnstructured(obj, &node); err != nil {
		return nil, err
	}

	for _, taint := range node.Spec.Taints {
		if taint.Key == "node.kubernetes.io/unschedulable" && taint.Effect == "NoSchedule" {
			return &HealthStatus{
				Ready:  false,
				Health: HealthWarning,
				Status: "Unschedulable",
			}, nil
		}
	}

	for _, cond := range node.Status.Conditions {
		if cond.Type == v1.NodeReady && cond.Status == v1.ConditionTrue {
			return &HealthStatus{
				Ready:  true,
				Health: HealthHealthy,
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

	return &HealthStatus{
		Status:  HealthStatusUnknown,
		Health:  HealthUnknown,
		Message: "no conditions matched for node status",
	}, nil
}
