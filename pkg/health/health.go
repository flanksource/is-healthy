package health

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Health string

const (
	HealthHealthy   Health = "healthy"
	HealthUnhealthy Health = "unhealthy"
	HealthUnknown   Health = "unknown"
	HealthWarning   Health = "warning"
)

// Represents resource health status
type HealthStatusCode string

const (
	// Indicates that health assessment failed and actual health status is unknown
	HealthStatusUnknown HealthStatusCode = "Unknown"
	// Progressing health status means that resource is not healthy but still have a chance to reach healthy state
	HealthStatusProgressing HealthStatusCode = "Progressing"
	// Resource is 100% healthy
	HealthStatusHealthy HealthStatusCode = "Healthy"
	// Assigned to resources that are suspended or paused. The typical example is a
	// [suspended](https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/#suspend) CronJob.
	HealthStatusSuspended HealthStatusCode = "Suspended"
	// Degrade status is used if resource status indicates failure or resource could not reach healthy state
	// within some timeout.
	HealthStatusDegraded HealthStatusCode = "Degraded"
	// Indicates that resource is missing in the cluster.
	HealthStatusMissing HealthStatusCode = "Missing"

	HealthStatusEvicted          HealthStatusCode = "Evicted"
	HealthStatusCompleted        HealthStatusCode = "Completed"
	HealthStatusCrashLoopBackoff HealthStatusCode = "CrashLoopBackOff"
	HealthStatusCreating         HealthStatusCode = "Creating"
	HealthStatusDeleted          HealthStatusCode = "Deleted"
	HealthStatusDeleting         HealthStatusCode = "Deleting"
	HealthStatusTerminating      HealthStatusCode = "Terminating"
	HealthStatusError            HealthStatusCode = "Error"
	HealthStatusRolloutFailed    HealthStatusCode = "Rollout Failed"
	HealthStatusInaccesible      HealthStatusCode = "Inaccesible"
	HealthStatusInfo             HealthStatusCode = "Info"
	HealthStatusPending          HealthStatusCode = "Pending"
	HealthStatusMaintenance      HealthStatusCode = "Maintenance"
	HealthStatusScaling          HealthStatusCode = "Scaling"
	HealthStatusRestart          HealthStatusCode = "Restarting"
	HealthStatusStarting         HealthStatusCode = "Starting"
	HealthStatusUnschedulable    HealthStatusCode = "Unschedulable"
	HealthStatusUpgradeFailed    HealthStatusCode = "UpgradeFailed"

	HealthStatusScalingUp    HealthStatusCode = "Scaling Up"
	HealthStatusScaledToZero HealthStatusCode = "Scaled to Zero"
	HealthStatusScalingDown  HealthStatusCode = "Scaling Down"
	HealthStatusRunning      HealthStatusCode = "Running"

	HealthStatusRollingOut HealthStatusCode = "Rolling Out"

	HealthStatusUnhealthy HealthStatusCode = "Unhealthy"
	HealthStatusUpdating  HealthStatusCode = "Updating"
	HealthStatusWarning   HealthStatusCode = "Warning"
	HealthStatusStopped   HealthStatusCode = "Stopped"
	HealthStatusStopping  HealthStatusCode = "Stopping"
)

// Implements custom health assessment that overrides built-in assessment
type HealthOverride interface {
	GetResourceHealth(obj *unstructured.Unstructured) (*HealthStatus, error)
}

// healthOrder is a list of health codes in order of most healthy to least healthy
var healthOrder = []HealthStatusCode{
	HealthStatusHealthy,
	HealthStatusSuspended,
	HealthStatusProgressing,
	HealthStatusMissing,
	HealthStatusDegraded,
	HealthStatusUnknown,
}

// IsWorse returns whether or not the new health status code is a worse condition than the current
func IsWorse(current, new HealthStatusCode) bool {
	currentIndex := 0
	newIndex := 0
	for i, code := range healthOrder {
		if current == code {
			currentIndex = i
		}
		if new == code {
			newIndex = i
		}
	}
	return newIndex > currentIndex
}

// GetResourceHealth returns the health of a k8s resource
func GetResourceHealth(obj *unstructured.Unstructured, healthOverride HealthOverride) (health *HealthStatus, err error) {
	if healthCheck := GetHealthCheckFunc(obj.GroupVersionKind()); healthCheck != nil {
		if health, err = healthCheck(obj); err != nil {
			health = &HealthStatus{
				Status:  HealthStatusUnknown,
				Message: err.Error(),
			}
		} else {
			return health, nil
		}
	}

	if healthOverride != nil {
		health, err := healthOverride.GetResourceHealth(obj)
		if err != nil {
			health = &HealthStatus{
				Status:  HealthStatusUnknown,
				Message: err.Error(),
			}
			return health, err
		}
		if health != nil {
			return health, nil
		}
	}

	if obj.GetDeletionTimestamp() != nil {
		return &HealthStatus{
			Status: HealthStatusTerminating,
		}, nil
	}

	if health == nil {
		return &HealthStatus{
			Status: HealthStatusUnknown,
			Ready:  true,
		}, nil
	}
	return health, err

}

// GetHealthCheckFunc returns built-in health check function or nil if health check is not supported
func GetHealthCheckFunc(gvk schema.GroupVersionKind) func(obj *unstructured.Unstructured) (*HealthStatus, error) {
	if gvk.Kind == "Node" {
		return getNodeHealth
	}

	if strings.HasSuffix(gvk.Group, ".crossplane.io") || strings.HasSuffix(gvk.Group, ".upbound.io") {
		return GetDefaultHealth
	}

	switch gvk.Group {
	case "apps":
		switch gvk.Kind {
		case DeploymentKind:
			return getDeploymentHealth
		case StatefulSetKind:
			return getStatefulSetHealth
		case ReplicaSetKind:
			return getReplicaSetHealth
		case DaemonSetKind:
			return getDaemonSetHealth
		}
	case "extensions":
		switch gvk.Kind {
		case IngressKind:
			return getIngressHealth
		}
	case "argoproj.io":
		switch gvk.Kind {
		case "Workflow":
			return GetArgoWorkflowHealth
		case "Application":
			return getArgoApplicationHealth
		}
	case "kustomize.toolkit.fluxcd.io", "helm.toolkit.fluxcd.io", "source.toolkit.fluxcd.io":
		return GetDefaultHealth
	case "cert-manager.io":
		return GetCertificateHealth
	case "networking.k8s.io":
		switch gvk.Kind {
		case IngressKind:
			return getIngressHealth
		}
	case "":
		switch gvk.Kind {
		case ServiceKind:
			return getServiceHealth
		case PersistentVolumeClaimKind:
			return getPVCHealth
		case PodKind:
			return getPodHealth
		case NamespaceKind:
			return getNamespaceHealth
		}
	case "batch":
		switch gvk.Kind {
		case JobKind:
			return getJobHealth
		case CronJobKind:
			return getCronJobHealth
		}
	case "autoscaling":
		switch gvk.Kind {
		case HorizontalPodAutoscalerKind:
			return getHPAHealth
		}
	}
	return nil
}
