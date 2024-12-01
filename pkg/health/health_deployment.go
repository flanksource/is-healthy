package health

import (
	"fmt"
	"time"

	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getDeploymentHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case appsv1.SchemeGroupVersion.WithKind(DeploymentKind):
		var deployment appsv1.Deployment
		err := convertFromUnstructured(obj, &deployment)
		if err != nil {
			return nil, err
		}
		return getAppsv1DeploymentHealth(&deployment, obj)
	default:
		return nil, fmt.Errorf("unsupported Deployment GVK: %s", gvk)
	}
}

type ReplicaStatus struct {
	Object                                         *unstructured.Unstructured
	Containers                                     []corev1.Container
	Desired, Replicas, Ready, Updated, Unavailable int
}

func (rs ReplicaStatus) String() string {
	s := fmt.Sprintf("%d/%d ready", rs.Ready, rs.Desired)

	if rs.Replicas != rs.Updated {
		s += fmt.Sprintf(", %d updating", rs.Replicas-rs.Updated)
	}

	if rs.Replicas > rs.Desired {
		s += fmt.Sprintf(", %d terminating", rs.Replicas-rs.Desired)
	}
	return s
}

func getReplicaHealth(s ReplicaStatus) *HealthStatus {
	hs := &HealthStatus{
		Message: s.String(),
	}
	startDeadline := GetStartDeadline(s.Containers...)
	age := time.Since(s.Object.GetCreationTimestamp().Time).Truncate(time.Minute).Abs()

	gs := GetGenericStatus(s.Object)

	available := gs.FindCondition("Available")
	isAvailable := s.Ready > 0
	if available.Status != "" {
		isAvailable = available.Status == "True"
	}

	progressing := gs.FindCondition("Progressing")

	failure := gs.FindCondition("ReplicaFailure")
	if failure.Status == "True" {
		hs.Status = HealthStatusFailedCreate
		hs.Health = HealthUnhealthy
		hs.Message = failure.Message
		hs.Ready = true
		return hs
	}

	isStarting := age < startDeadline
	isProgressDeadlineExceeded := !isStarting && (progressing.Reason == "ProgressDeadlineExceeded")
	hs.Ready = progressing.Status == "True" && progressing.Reason != "ReplicaSetUpdated"

	hs.Health = lo.Ternary(isAvailable, HealthHealthy, lo.Ternary(s.Ready > 0, HealthWarning, HealthUnhealthy))

	if s.Desired == 0 && s.Replicas == 0 {
		hs.Ready = true
		hs.Status = HealthStatusScaledToZero
		hs.Health = HealthUnknown
		return hs
	}
	if s.Replicas == 0 {
		if isProgressDeadlineExceeded {
			hs.Status = "Failed Create"
			hs.Health = HealthUnhealthy
		} else {
			hs.Status = "Pending"
			hs.Health = HealthUnknown
		}
	} else if s.Ready == 0 && isStarting {
		hs.Status = HealthStatusStarting
	} else if s.Ready == 0 {
		if isProgressDeadlineExceeded {
			hs.Status = HealthStatusCrashLoopBackoff
		} else if isAvailable {
			hs.Status = HealthStatusUpdating
		}
	}

	if isProgressDeadlineExceeded {
		hs.Status = HealthStatusRolloutFailed
		hs.Health = hs.Health.Worst(HealthWarning)
	} else if s.Desired == 0 && s.Replicas > 0 {
		hs.Status = HealthStatusScalingDown
	} else if s.Ready == s.Desired && s.Desired == s.Updated && s.Replicas == s.Desired {
		hs.Status = HealthStatusRunning
	} else if !isStarting && s.Desired != s.Updated {
		hs.Status = HealthStatusRollingOut
	} else if s.Replicas > s.Desired {
		hs.Status = HealthStatusScalingDown
	} else if s.Replicas < s.Desired {
		hs.Status = HealthStatusScalingUp
	}

	if isStarting && (hs.Health == HealthUnhealthy || hs.Health == HealthWarning) {
		hs.Health = HealthUnknown
	}

	return hs
}

func getAppsv1DeploymentHealth(deployment *appsv1.Deployment, obj *unstructured.Unstructured) (*HealthStatus, error) {
	replicas := int32(0)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	replicaHealth := getReplicaHealth(
		ReplicaStatus{
			Object:     obj,
			Containers: deployment.Spec.Template.Spec.Containers,
			Desired:    int(replicas), Replicas: int(deployment.Status.Replicas),
			Ready: int(deployment.Status.ReadyReplicas), Updated: int(deployment.Status.UpdatedReplicas),
			Unavailable: int(deployment.Status.UnavailableReplicas),
		})

	if deployment.Spec.Paused {
		replicaHealth.Status = HealthStatusSuspended
		replicaHealth.Ready = false
	}

	return replicaHealth, nil
}
