package health

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getPVCHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case corev1.SchemeGroupVersion.WithKind(PersistentVolumeClaimKind):
		var pvc corev1.PersistentVolumeClaim
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &pvc)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured PersistentVolumeClaim to typed: %v", err)
		}
		return getCorev1PVCHealth(&pvc)
	default:
		return nil, fmt.Errorf("unsupported PersistentVolumeClaim GVK: %s", gvk)
	}
}

func getCorev1PVCHealth(pvc *corev1.PersistentVolumeClaim) (*HealthStatus, error) {
	health := HealthStatus{Health: HealthHealthy}
	switch pvc.Status.Phase {
	case corev1.ClaimLost:
		health.Health = HealthUnhealthy
		health.Status = HealthStatusDegraded
	case corev1.ClaimPending:
		health.Status = HealthStatusProgressing
	case corev1.ClaimBound:
		health.Ready = true
		health.Status = HealthStatusHealthy
	default:
		health.Health = HealthUnknown
		health.Status = HealthStatusUnknown
	}

	return &health, nil
}
