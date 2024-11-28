package health

import (
	"fmt"
	"time"

	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getServiceHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case corev1.SchemeGroupVersion.WithKind(ServiceKind):
		var service corev1.Service
		err := convertFromUnstructured(obj, &service)
		if err != nil {
			return nil, err
		}
		return getCorev1ServiceHealth(&service)
	default:
		return nil, fmt.Errorf("unsupported Service GVK: %s", gvk)
	}
}

func getCorev1ServiceHealth(service *corev1.Service) (*HealthStatus, error) {
	health := HealthStatus{Health: HealthHealthy, Status: HealthStatusHealthy}
	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			health.Status = HealthStatusRunning
			health.Health = HealthHealthy
			health.Ready = true
		} else {
			age := time.Since(service.CreationTimestamp.Time)
			health.Status = HealthStatusCreating
			health.Health = lo.Ternary(age < time.Hour, HealthUnknown, HealthUnhealthy)
		}
	} else {
		health.Ready = true
		health.Status = HealthStatusUnknown
		health.Health = HealthUnknown
	}
	return &health, nil
}
