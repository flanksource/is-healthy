package health

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getJobHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case batchv1.SchemeGroupVersion.WithKind(JobKind):
		var job batchv1.Job
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &job)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured Job to typed: %v", err)
		}
		return getBatchv1JobHealth(&job)
	default:
		return nil, fmt.Errorf("unsupported Job GVK: %s", gvk)
	}
}

func getBatchv1JobHealth(job *batchv1.Job) (*HealthStatus, error) {

	for _, condition := range job.Status.Conditions {
		switch condition.Type {
		case batchv1.JobFailed:
			return &HealthStatus{
				Ready:   true,
				Health:  HealthUnhealthy,
				Status:  HealthStatusError,
				Message: condition.Message,
			}, nil
		case batchv1.JobComplete:
			return &HealthStatus{
				Ready:   true,
				Status:  HealthStatusCompleted,
				Health:  HealthHealthy,
				Message: condition.Message,
			}, nil
		case batchv1.JobSuspended:
			if condition.Status == corev1.ConditionTrue {
				return &HealthStatus{
					Health:  HealthUnknown,
					Status:  HealthStatusSuspended,
					Message: condition.Message,
				}, nil
			}
		}
	}

	return &HealthStatus{
		Health: HealthHealthy,
		Status: HealthStatusRunning,
	}, nil
}
