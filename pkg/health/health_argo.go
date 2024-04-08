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

// An agnostic workflow object only considers Status.Phase and Status.Message. It is agnostic to the API version or any
// other fields.
type argoApplication struct {
	Status struct {
		Health HealthStatus
	}
}

func getArgoApplicationHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var app argoApplication
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &app); err != nil {
		return nil, err
	}

	switch app.Status.Health.Status {
	case HealthStatusProgressing:
		return &HealthStatus{Health: HealthHealthy, Status: HealthStatusProgressing, Message: app.Status.Health.Message}, nil
	case HealthStatusHealthy:
		return &HealthStatus{Ready: true, Health: HealthHealthy, Status: HealthStatusHealthy, Message: app.Status.Health.Message}, nil
	case HealthStatusSuspended:
		return &HealthStatus{Health: HealthHealthy, Status: HealthStatusSuspended, Message: app.Status.Health.Message}, nil
	case HealthStatusDegraded:
		return &HealthStatus{Health: HealthUnhealthy, Status: HealthStatusDegraded, Message: app.Status.Health.Message}, nil
	case HealthStatusMissing:
		return &HealthStatus{Health: HealthUnhealthy, Status: HealthStatusMissing, Message: app.Status.Health.Message}, nil
	case HealthStatusUnknown:
		return &HealthStatus{Health: HealthUnknown, Status: HealthStatusUnknown, Message: app.Status.Health.Message}, nil
	}

	return &HealthStatus{Health: HealthUnknown, Status: HealthStatusUnknown, Message: app.Status.Health.Message}, nil
}
