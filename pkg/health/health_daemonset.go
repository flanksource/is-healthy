package health

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getDaemonSetHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case appsv1.SchemeGroupVersion.WithKind(DaemonSetKind):
		var daemon appsv1.DaemonSet
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &daemon)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured DaemonSet to typed: %v", err)
		}
		return getAppsv1DaemonSetHealth(&daemon)
	default:
		return nil, fmt.Errorf("unsupported DaemonSet GVK: %s", gvk)
	}
}

func getAppsv1DaemonSetHealth(daemon *appsv1.DaemonSet) (*HealthStatus, error) {
	health := HealthUnknown

	if daemon.Status.NumberAvailable == daemon.Status.DesiredNumberScheduled {
		health = HealthHealthy
	} else if daemon.Status.NumberAvailable > 0 {
		health = HealthWarning
	} else if daemon.Status.NumberAvailable == 0 {
		health = HealthUnhealthy
	}

	if daemon.Generation == daemon.Status.ObservedGeneration &&
		daemon.Status.UpdatedNumberScheduled == daemon.Status.DesiredNumberScheduled {
		return &HealthStatus{
			Health: HealthHealthy,
			Ready:  true,
			Status: HealthStatusRunning,
		}, nil
	}

	if daemon.Spec.UpdateStrategy.Type == appsv1.OnDeleteDaemonSetStrategyType {
		return &HealthStatus{
			Health: health,
			Ready:  daemon.Status.NumberAvailable == daemon.Status.DesiredNumberScheduled,
			Status: HealthStatusRunning,
			Message: fmt.Sprintf(
				"%d of %d pods updated",
				daemon.Status.UpdatedNumberScheduled,
				daemon.Status.DesiredNumberScheduled,
			),
		}, nil
	}
	if daemon.Status.UpdatedNumberScheduled < daemon.Status.DesiredNumberScheduled {
		return &HealthStatus{
			Health: health,
			Status: HealthStatusRollingOut,
			Message: fmt.Sprintf(
				"%d of %d pods updated",
				daemon.Status.UpdatedNumberScheduled,
				daemon.Status.DesiredNumberScheduled,
			),
		}, nil
	}
	if daemon.Status.NumberAvailable < daemon.Status.DesiredNumberScheduled {
		return &HealthStatus{
			Health: health,
			Status: HealthStatusRollingOut,
			Message: fmt.Sprintf(
				"%d of %d pods ready",
				daemon.Status.NumberAvailable,
				daemon.Status.DesiredNumberScheduled,
			),
		}, nil
	}

	return &HealthStatus{
		Status: HealthStatusRunning,
		Health: health,
	}, nil
}
