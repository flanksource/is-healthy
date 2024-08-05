package health

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getDeploymentHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case appsv1.SchemeGroupVersion.WithKind(DeploymentKind):
		var deployment appsv1.Deployment
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deployment)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured Deployment to typed: %v", err)
		}
		return getAppsv1DeploymentHealth(&deployment, obj)
	default:
		return nil, fmt.Errorf("unsupported Deployment GVK: %s", gvk)
	}
}

func getAppsv1DeploymentHealth(deployment *appsv1.Deployment, obj *unstructured.Unstructured) (*HealthStatus, error) {
	status, err := GetDefaultHealth(obj)
	if err != nil {
		return status, err
	}

	replicas := int32(0)

	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	if replicas == 0 && deployment.Status.Replicas == 0 {
		return &HealthStatus{
			Ready:  true,
			Status: HealthStatusScaledToZero,
			Health: HealthUnknown,
		}, nil
	}

	if deployment.Status.ReadyReplicas == replicas {
		status.PrependMessage("%d pods ready", deployment.Status.ReadyReplicas)
	} else {
		status.PrependMessage("%d of %d pods ready", deployment.Status.ReadyReplicas, replicas)
	}

	if deployment.Spec.Paused {
		status.Ready = false
		status.Status = HealthStatusSuspended
		return status, err
	}

	if deployment.Status.ReadyReplicas > 0 {
		status.Status = HealthStatusRunning
	}

	if status.Health == HealthUnhealthy {
		return status, nil
	}

	if deployment.Status.ReadyReplicas < replicas {
		status.AppendMessage("%d starting", deployment.Status.Replicas-deployment.Status.ReadyReplicas)
		if deployment.Status.Replicas < replicas {
			status.AppendMessage("%d creating", replicas-deployment.Status.Replicas)
		}
		status.Ready = false
		status.Status = HealthStatusStarting
	} else if deployment.Status.UpdatedReplicas < replicas {
		status.AppendMessage("%d updating", replicas-deployment.Status.UpdatedReplicas)
		status.Ready = false
		status.Status = HealthStatusRollingOut
	} else if deployment.Status.Replicas > replicas {
		status.AppendMessage("%d pods terminating", deployment.Status.Replicas-replicas)
		status.Ready = false
		status.Status = HealthStatusScalingDown
	}

	return status, nil
}
