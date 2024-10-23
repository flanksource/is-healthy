package health

import (
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getStatefulSetHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case appsv1.SchemeGroupVersion.WithKind(StatefulSetKind):
		var sts appsv1.StatefulSet
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &sts)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured StatefulSet to typed: %v", err)
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
		}, nil
	}

	var containersWaitingForReadiness []string
	for _, container := range sts.Spec.Template.Spec.Containers {
		if container.ReadinessProbe != nil && container.ReadinessProbe.InitialDelaySeconds > 0 {
			deadline := sts.CreationTimestamp.Add(
				time.Second * time.Duration(container.ReadinessProbe.InitialDelaySeconds),
			)
			if time.Now().Before(deadline) {
				containersWaitingForReadiness = append(containersWaitingForReadiness, container.Name)
			}
		}
	}

	if len(containersWaitingForReadiness) > 0 {
		return &HealthStatus{
			Health: HealthUnknown,
			Status: HealthStatusStarting,
			Message: fmt.Sprintf(
				"Container(s) %s is waiting for readiness probe",
				strings.Join(containersWaitingForReadiness, ","),
			),
		}, nil
	}

	health := HealthHealthy
	if sts.Status.ReadyReplicas == 0 {
		health = HealthUnhealthy
	} else if sts.Status.UpdatedReplicas == 0 {
		health = HealthWarning
	} else if sts.Spec.Replicas != nil && sts.Status.ReadyReplicas >= *sts.Spec.Replicas {
		health = HealthHealthy
	}

	if sts.Spec.Replicas != nil && sts.Status.ReadyReplicas < *sts.Spec.Replicas {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusRollingOut,
			Message: fmt.Sprintf("%d of %d pods ready", sts.Status.ReadyReplicas, *sts.Spec.Replicas),
		}, nil
	}

	if sts.Spec.Replicas != nil && sts.Status.UpdatedReplicas < *sts.Spec.Replicas {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusRollingOut,
			Message: fmt.Sprintf("%d of %d pods updated", sts.Status.UpdatedReplicas, *sts.Spec.Replicas),
		}, nil
	}

	if sts.Status.ObservedGeneration == 0 || sts.Generation > sts.Status.ObservedGeneration {
		return &HealthStatus{
			Health: health,
			Status: HealthStatusRollingOut,
		}, nil
	}

	return &HealthStatus{
		Ready:  true,
		Health: health,
		Status: HealthStatusRunning,
	}, nil

}
