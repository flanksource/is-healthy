package health

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type nodePhase string

// Workflow and node statuses
// See: https://github.com/argoproj/argo-workflows/blob/master/pkg/apis/workflow/v1alpha1/workflow_phase.go
const (
	nodePending   nodePhase = "Pending"
	nodeRunning   nodePhase = "Running"
	nodeSucceeded nodePhase = "Succeeded"
	nodeFailed    nodePhase = "Failed"
	nodeError     nodePhase = "Error"
)

func GetArgoWorkflowHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var phase nodePhase
	var message string

	if status, ok := obj.Object["status"].(map[string]interface{}); ok {
		if p, ok := status["phase"].(string); ok {
			phase = nodePhase(p)
		}
		if m, ok := status["message"].(string); ok {
			message = m
		}
	}

	switch phase {
	case "", nodePending:
		return &HealthStatus{Health: HealthHealthy, Status: HealthStatusProgressing, Message: message}, nil
	case nodeRunning:
		return &HealthStatus{
			Ready:   true,
			Health:  HealthHealthy,
			Status:  HealthStatusProgressing,
			Message: message,
		}, nil
	case nodeSucceeded:
		return &HealthStatus{
			Ready:   true,
			Health:  HealthHealthy,
			Status:  HealthStatusHealthy,
			Message: message,
		}, nil
	case nodeFailed, nodeError:
		return &HealthStatus{Health: HealthUnhealthy, Status: HealthStatusDegraded, Message: message}, nil
	}
	return &HealthStatus{Health: HealthUnknown, Status: HealthStatusUnknown, Message: message}, nil
}

const (
	SyncStatusCodeSynced = "Synced"
)

func getArgoApplicationHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	hs := &HealthStatus{Health: HealthUnknown}
	var status map[string]interface{}

	status, ok := obj.Object["status"].(map[string]interface{})
	if !ok {
		return hs, nil
	}

	if sync, ok := status["sync"].(map[string]interface{}); ok {
		hs.Ready = sync["status"] == SyncStatusCodeSynced
	}
	if health, ok := status["health"].(map[string]interface{}); ok {
		if message, ok := health["message"]; ok {
			hs.Message = message.(string)
		}
		if argoHealth, ok := health["status"]; ok {
			hs.Status = HealthStatusCode(argoHealth.(string))
			switch hs.Status {
			case HealthStatusHealthy:
				hs.Health = HealthHealthy
			case HealthStatusDegraded:
				hs.Health = HealthUnhealthy
			case HealthStatusUnknown, HealthStatusMissing, HealthStatusProgressing, HealthStatusSuspended:
				hs.Health = HealthUnknown
			}
		}
	}
	return hs, nil
}
