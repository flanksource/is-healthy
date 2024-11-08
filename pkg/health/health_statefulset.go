package health

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getStatefulSetHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case appsv1.SchemeGroupVersion.WithKind(StatefulSetKind):
		var sts appsv1.StatefulSet
		if err := convertFromUnstructured(obj, &sts); err != nil {
			return nil, err
		}
		return getAppsv1StatefulSetHealth(&sts)
	default:
		return nil, fmt.Errorf("unsupported StatefulSet GVK: %s", gvk)
	}
}

func getAppsv1StatefulSetHealth(sts *appsv1.StatefulSet) (*HealthStatus, error) {
	replicas := int32(0)
	if sts.Spec.Replicas != nil {
		replicas = *sts.Spec.Replicas
	}

	if replicas == 0 && sts.Status.Replicas == 0 {
		return &HealthStatus{
			Status: HealthStatusScaledToZero,
			Health: HealthUnknown,
			Ready:  true,
		}, nil
	}

	startDeadline := GetStartDeadline(sts.Spec.Template.Spec.Containers...)
	age := time.Since(sts.CreationTimestamp.Time).Truncate(time.Minute).Abs()

	health := HealthHealthy
	if sts.Status.ReadyReplicas == 0 {
		if sts.Status.CurrentReplicas > 0 && age < startDeadline {
			health = HealthUnknown
		} else {
			health = HealthUnhealthy
		}
	} else if sts.Status.UpdatedReplicas == 0 {
		health = HealthWarning
	} else if sts.Status.ReadyReplicas >= replicas {
		health = HealthHealthy
	}

	if sts.Spec.Replicas != nil && sts.Status.ReadyReplicas < *sts.Spec.Replicas {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusStarting,
			Message: fmt.Sprintf("%d of %d pods ready", sts.Status.ReadyReplicas, *sts.Spec.Replicas),
		}, nil
	}

	if sts.Spec.Replicas != nil && sts.Status.UpdatedReplicas < replicas {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusRollingOut,
			Message: fmt.Sprintf("%d of %d pods updated, %d of %d ready", sts.Status.UpdatedReplicas, replicas, sts.Status.ReadyReplicas, replicas),
		}, nil
	}

	if sts.Status.ObservedGeneration == 0 || sts.Generation > sts.Status.ObservedGeneration {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusRollingOut,
			Message: fmt.Sprintf("generation not up to date %d", sts.Generation),
		}, nil
	}

	if sts.Status.UpdateRevision != "" && sts.Status.CurrentRevision != sts.Status.UpdateRevision {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusRollingOut,
			Message: fmt.Sprintf("revision not up to date %s", sts.Status.UpdateRevision),
		}, nil
	}

	return &HealthStatus{
		Ready:  true,
		Health: health,
		Status: HealthStatusRunning,
	}, nil
}
