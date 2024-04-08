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
	var (
		complete    bool
		failed      bool
		isSuspended bool

		message string
		failMsg string
	)

	for _, condition := range job.Status.Conditions {
		switch condition.Type {
		case batchv1.JobFailed:
			failed = true
			complete = true
			failMsg = condition.Message
		case batchv1.JobComplete:
			complete = true
			message = condition.Message
		case batchv1.JobSuspended:
			message = condition.Message
			if condition.Status == corev1.ConditionTrue {
				isSuspended = true
			}
		}
	}

	if !complete && !isSuspended {
		return &HealthStatus{
			Health:  HealthHealthy,
			Status:  HealthStatusProgressing,
			Message: message,
		}, nil
	} else if failed {
		return &HealthStatus{
			Health:  HealthUnhealthy,
			Status:  HealthStatusDegraded,
			Message: failMsg,
		}, nil
	} else if isSuspended {
		return &HealthStatus{
			Health:  HealthHealthy,
			Status:  HealthStatusSuspended,
			Message: failMsg,
		}, nil
	} else {
		return &HealthStatus{
			Ready:   true,
			Status:  HealthStatusHealthy,
			Health:  HealthHealthy,
			Message: message,
		}, nil
	}
}
