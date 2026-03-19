package health

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getDaemonSetHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case appsv1.SchemeGroupVersion.WithKind(DaemonSetKind):
		var daemon appsv1.DaemonSet
		err := convertFromUnstructured(obj, &daemon)
		if err != nil {
			return nil, err
		}
		return getAppsv1DaemonSetHealth(&daemon)
	default:
		return nil, fmt.Errorf("unsupported DaemonSet GVK: %s", gvk)
	}
}

func getAppsv1DaemonSetHealth(daemon *appsv1.DaemonSet) (*HealthStatus, error) {
	desired := daemon.Status.DesiredNumberScheduled
	updated := daemon.Status.UpdatedNumberScheduled
	available := daemon.Status.NumberAvailable
	ready := daemon.Status.NumberReady
	unavailable := daemon.Status.NumberUnavailable

	fullyUpdated := updated == desired
	fullyAvailable := desired == 0 || (available == desired && ready == desired)

	var health Health
	switch {
	case desired == 0:
		health = HealthUnknown
	case fullyAvailable:
		health = HealthHealthy
	case available > 0 || ready > 0:
		health = HealthWarning
	default:
		health = HealthUnhealthy
	}

	if daemon.Spec.UpdateStrategy.Type == appsv1.OnDeleteDaemonSetStrategyType {
		return &HealthStatus{
			Health:  health,
			Ready:   fullyAvailable,
			Status:  HealthStatusRunning,
			Message: fmt.Sprintf("%d of %d pods updated", updated, desired),
		}, nil
	}

	if !fullyUpdated {
		return &HealthStatus{
			Health:  health,
			Ready:   false,
			Status:  HealthStatusRollingOut,
			Message: fmt.Sprintf("%d of %d pods updated", updated, desired),
		}, nil
	}

	if !fullyAvailable {
		replicaWord := "replica"
		if unavailable != 1 {
			replicaWord = "replicas"
		}

		return &HealthStatus{
			Health: health,
			Ready:  false,
			Status: HealthStatusPartiallyAvailable,
			Message: fmt.Sprintf(
				"DaemonSet not fully available: %d unavailable %s (%d/%d available, %d/%d ready)\n",
				unavailable,
				replicaWord,
				available,
				desired,
				ready,
				desired,
			),
		}, nil
	}

	return &HealthStatus{
		Health: health,
		Ready:  true,
		Status: HealthStatusRunning,
	}, nil
}
