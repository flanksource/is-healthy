package health

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getCronJobHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case batchv1.SchemeGroupVersion.WithKind(CronJobKind):
		var job batchv1.CronJob
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &job)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured Job to typed: %v", err)
		}
		return getBatchv1CronJobHealth(&job)
	default:
		return nil, fmt.Errorf("unsupported CronJob GVK: %s", gvk)
	}
}

func getBatchv1CronJobHealth(job *batchv1.CronJob) (*HealthStatus, error) {
	if job.Status.LastScheduleTime == nil {
		return &HealthStatus{
			Health:  HealthUnknown,
			Message: "Not scheduled yet",
		}, nil
	}

	if job.Status.LastSuccessfulTime == nil {
		return &HealthStatus{
			Health:  HealthUnhealthy,
			Status:  HealthStatusError,
			Message: "No successful run yet",
		}, nil
	}

	if len(job.Status.Active) > 0 {
		return &HealthStatus{
			Health:  HealthHealthy,
			Status:  HealthStatusRunning,
			Message: "Running since" + job.Status.LastScheduleTime.Format("2006-01-02 15:04:05 -0700"),
		}, nil
	}

	if job.Status.LastSuccessfulTime.Before(job.Status.LastScheduleTime) {
		return &HealthStatus{
			Ready:   true, // The cronjob did in fact run
			Health:  HealthUnhealthy,
			Status:  HealthStatusError,
			Message: "Last run failed, last successful run was" + job.Status.LastSuccessfulTime.Format("2006-01-02 15:04:05 -0700"),
		}, nil
	}

	return &HealthStatus{
		Ready:   true,
		Health:  HealthHealthy,
		Status:  HealthStatusCompleted,
		Message: fmt.Sprintf("Last run at %s in %s", job.Status.LastScheduleTime.Format("2006-01-02 15:04:05 -0700"), job.Status.LastSuccessfulTime.Sub(job.Status.LastScheduleTime.Time)),
	}, nil
}
