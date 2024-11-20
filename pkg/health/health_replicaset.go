package health

import (
	"fmt"

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
		return getAppsv1ReplicaSetHealth(&replicaSet, obj)
	default:
		return nil, fmt.Errorf("unsupported ReplicaSet GVK: %s", gvk)
	}
}

func getAppsv1ReplicaSetHealth(rs *appsv1.ReplicaSet, obj *unstructured.Unstructured) (*HealthStatus, error) {
	replicas := int32(0)
	if rs.Spec.Replicas != nil {
		replicas = *rs.Spec.Replicas
	}

	hr := getReplicaHealth(ReplicaStatus{
		Object:     obj,
		Containers: rs.Spec.Template.Spec.Containers,
		Desired:    int(replicas),
		Replicas:   int(rs.Status.Replicas),
		Ready:      int(rs.Status.ReadyReplicas),
		Updated:    int(rs.Status.FullyLabeledReplicas),
	})

	if rs.Generation != rs.Status.ObservedGeneration {
		hr.Status = HealthStatusUpdating
		hr.Ready = false
	}

	failCondition := getAppsv1ReplicaSetCondition(rs.Status, appsv1.ReplicaSetReplicaFailure)
	if hr.Health != HealthUnhealthy && failCondition != nil && failCondition.Status == corev1.ConditionTrue {
		hr.Ready = true
		hr.Health = HealthUnhealthy
		hr.Status = HealthStatusCode(HumanCase(failCondition.Reason))
		hr.Message = failCondition.Message
	}
	return hr, nil
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
