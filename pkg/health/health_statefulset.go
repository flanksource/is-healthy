package health

import (
	"fmt"

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
		return getAppsv1StatefulSetHealth(&sts, obj)
	default:
		return nil, fmt.Errorf("unsupported StatefulSet GVK: %s", gvk)
	}
}

func getAppsv1StatefulSetHealth(sts *appsv1.StatefulSet, obj *unstructured.Unstructured) (*HealthStatus, error) {
	replicas := int32(0)
	if sts.Spec.Replicas != nil {
		replicas = *sts.Spec.Replicas
	}

	replicaHealth := getReplicaHealth(
		ReplicaStatus{
			Object:     obj,
			Containers: sts.Spec.Template.Spec.Containers,
			Desired:    int(replicas), Replicas: int(sts.Status.Replicas),
			Ready: int(sts.Status.ReadyReplicas), Updated: int(sts.Status.UpdatedReplicas),
		})

	replicaHealth.Ready = sts.Status.Replicas == sts.Status.UpdatedReplicas

	return replicaHealth, nil
}
