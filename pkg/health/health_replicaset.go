package health

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getReplicaSetHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case appsv1.SchemeGroupVersion.WithKind(ReplicaSetKind):
		var replicaSet appsv1.ReplicaSet
		err := convertFromUnstructured(obj, &replicaSet)
		if err != nil {
			return nil, err
		}
		return getAppsv1ReplicaSetHealth(&replicaSet)
	default:
		return nil, fmt.Errorf("unsupported ReplicaSet GVK: %s", gvk)
	}
}

func getAppsv1ReplicaSetHealth(rs *appsv1.ReplicaSet) (*HealthStatus, error) {
	replicas := int32(0)
	if rs.Spec.Replicas != nil {
		replicas = *rs.Spec.Replicas
	}
	startDeadline := GetStartDeadline(rs.Spec.Template.Spec.Containers...)
	age := time.Since(rs.CreationTimestamp.Time).Truncate(time.Minute).Abs()

	health := HealthHealthy
	if rs.Status.ReadyReplicas == 0 {
		if rs.Status.Replicas > 0 && age < startDeadline {
			health = HealthUnknown
		} else {
			health = HealthUnhealthy
		}
	} else if rs.Status.ReadyReplicas < replicas {
		health = HealthWarning
	} else if rs.Status.ReadyReplicas >= replicas {
		health = HealthHealthy
	}

	if replicas == 0 && rs.Status.Replicas == 0 {
		return &HealthStatus{
			Ready:  true,
			Status: HealthStatusScaledToZero,
			Health: health,
		}, nil
	}

	if rs.Generation == rs.Status.ObservedGeneration &&
		rs.Status.ReadyReplicas == *rs.Spec.Replicas {
		return &HealthStatus{
			Health: health,
			Status: HealthStatusRunning,
			Ready:  true,
		}, nil
	}

	failCondition := getAppsv1ReplicaSetCondition(rs.Status, appsv1.ReplicaSetReplicaFailure)
	if failCondition != nil && failCondition.Status == corev1.ConditionTrue {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusError,
			Message: failCondition.Message,
		}, nil
	}

	if rs.Status.ReadyReplicas < *rs.Spec.Replicas {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusScalingUp,
			Message: fmt.Sprintf("%d of %d pods ready", rs.Status.ReadyReplicas, *rs.Spec.Replicas),
		}, nil
	}

	if rs.Status.ReadyReplicas > *rs.Spec.Replicas {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusScalingDown,
			Message: fmt.Sprintf("%d pods terminating", rs.Status.ReadyReplicas-*rs.Spec.Replicas),
		}, nil
	}

	return &HealthStatus{
		Status: HealthStatusUnknown,
		Health: health,
	}, nil
}

func getAppsv1ReplicaSetCondition(
	status appsv1.ReplicaSetStatus,
	condType appsv1.ReplicaSetConditionType,
) *appsv1.ReplicaSetCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}
