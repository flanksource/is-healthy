package health

import (
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// duration after the creation of a replica set
// within which we never deem the it to be unhealthy.
const replicaSetBufferPeriod = time.Minute * 10

func getReplicaSetHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case appsv1.SchemeGroupVersion.WithKind(ReplicaSetKind):
		var replicaSet appsv1.ReplicaSet
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &replicaSet)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured ReplicaSet to typed: %v", err)
		}
		return getAppsv1ReplicaSetHealth(&replicaSet)
	default:
		return nil, fmt.Errorf("unsupported ReplicaSet GVK: %s", gvk)
	}
}

func getAppsv1ReplicaSetHealth(replicaSet *appsv1.ReplicaSet) (*HealthStatus, error) {
	isWithinBufferPeriod := replicaSet.CreationTimestamp.Add(replicaSetBufferPeriod).After(time.Now())

	var containersWaitingForReadiness []string
	for _, container := range replicaSet.Spec.Template.Spec.Containers {
		if container.ReadinessProbe != nil && container.ReadinessProbe.InitialDelaySeconds > 0 {
			deadline := replicaSet.CreationTimestamp.Add(
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

	health := HealthUnknown
	if (replicaSet.Spec.Replicas == nil || *replicaSet.Spec.Replicas == 0) && replicaSet.Status.Replicas == 0 {
		return &HealthStatus{
			Ready:  true,
			Status: HealthStatusScaledToZero,
			Health: health,
		}, nil
	}

	if replicaSet.Spec.Replicas != nil && replicaSet.Status.ReadyReplicas >= *replicaSet.Spec.Replicas {
		health = HealthHealthy
	} else if replicaSet.Status.ReadyReplicas > 0 {
		health = HealthWarning
	} else {
		health = HealthUnhealthy
	}

	if (health == HealthUnhealthy || health == HealthWarning) && isWithinBufferPeriod {
		// within the buffer period, we don't mark a ReplicaSet as unhealthy
		health = HealthUnknown
	}

	if replicaSet.Generation == replicaSet.Status.ObservedGeneration &&
		replicaSet.Status.ReadyReplicas == *replicaSet.Spec.Replicas {
		return &HealthStatus{
			Health: health,
			Status: HealthStatusRunning,
			Ready:  true,
		}, nil
	}

	failCondition := getAppsv1ReplicaSetCondition(replicaSet.Status, appsv1.ReplicaSetReplicaFailure)
	if failCondition != nil && failCondition.Status == corev1.ConditionTrue {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusError,
			Message: failCondition.Message,
		}, nil
	}

	if replicaSet.Status.ReadyReplicas < *replicaSet.Spec.Replicas {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusScalingUp,
			Message: fmt.Sprintf("%d of %d pods ready", replicaSet.Status.ReadyReplicas, *replicaSet.Spec.Replicas),
		}, nil
	}

	if replicaSet.Status.ReadyReplicas > *replicaSet.Spec.Replicas {
		return &HealthStatus{
			Health:  health,
			Status:  HealthStatusScalingDown,
			Message: fmt.Sprintf("%d pods terminating", replicaSet.Status.ReadyReplicas-*replicaSet.Spec.Replicas),
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
