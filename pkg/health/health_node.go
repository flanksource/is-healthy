package health

import (
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getNodeHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var node v1.Node
	if err := convertFromUnstructured(obj, &node); err != nil {
		return nil, err
	}

	hs := HealthStatus{
		Status: HealthStatusCode(node.Status.Phase),
		Health: HealthUnknown,
	}
	switch node.Status.Phase {
	case v1.NodeRunning, "":
		for _, cond := range node.Status.Conditions {
			if cond.Type == v1.NodeReady {
				if cond.Status == v1.ConditionTrue {
					hs.Ready = true
					hs.Status = lo.CoalesceOrEmpty(hs.Status, HealthStatusRunning)
					hs.Health = hs.Health.Worst(HealthHealthy)
				} else {
					hs.Health = HealthUnhealthy
					hs.Status = HealthStatusCode(HumanCase(string(cond.Type)))
					hs.Message = cond.Message
				}
			} else if cond.Status == v1.ConditionTrue && cond.Type != "SysctlChanged" {
				hs.Health = (HealthWarning)
				hs.Status = HealthStatusCode(HumanCase(string(cond.Type)))
				hs.Message = cond.Message
			}
		}
		for _, taint := range node.Spec.Taints {
			if taint.Key == "node.kubernetes.io/unschedulable" && taint.Effect == "NoSchedule" {
				hs.Health = hs.Health.Worst(HealthWarning)
				hs.Status = "Unschedulable"
			}
		}
	}

	return &hs, nil
}
