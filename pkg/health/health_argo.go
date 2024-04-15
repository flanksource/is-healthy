package health

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

// An agnostic workflow object only considers Status.Phase and Status.Message. It is agnostic to the API version or any
// other fields.
type argoWorkflow struct {
	Status struct {
		Phase   nodePhase
		Message string
	}
}

func GetArgoWorkflowHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var wf argoWorkflow
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &wf)
	if err != nil {
		return nil, err
	}
	switch wf.Status.Phase {
	case "", nodePending:
		return &HealthStatus{Health: HealthHealthy, Status: HealthStatusProgressing, Message: wf.Status.Message}, nil
	case nodeRunning:
		return &HealthStatus{Ready: true, Health: HealthHealthy, Status: HealthStatusProgressing, Message: wf.Status.Message}, nil
	case nodeSucceeded:
		return &HealthStatus{Ready: true, Health: HealthHealthy, Status: HealthStatusHealthy, Message: wf.Status.Message}, nil
	case nodeFailed, nodeError:
		return &HealthStatus{Health: HealthUnhealthy, Status: HealthStatusDegraded, Message: wf.Status.Message}, nil
	}
	return &HealthStatus{Health: HealthUnknown, Status: HealthStatusUnknown, Message: wf.Status.Message}, nil
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
