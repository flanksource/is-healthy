package health

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/properties"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/duration"
)

var DefaultOverrides HealthOverride

type Health string

const (
	HealthHealthy   Health = "healthy"
	HealthUnhealthy Health = "unhealthy"
	HealthUnknown   Health = "unknown"
	HealthWarning   Health = "warning"
)

func IsValidHealth(s string) bool {
	return s == string(HealthHealthy) || s == string(HealthUnhealthy) || s == string(HealthUnknown) ||
		s == string(HealthWarning)
}

var healthOrder = []Health{
	HealthUnknown,
	HealthHealthy,
	HealthWarning,
	HealthUnhealthy,
}

func (h Health) Worst(others ...Health) Health {
	all := append(others, h)
	slices.SortFunc(all, CompareHealth)
	return all[len(all)-1]
}

func (h Health) IsWorseThan(other Health) bool {
	return h.CompareTo(other) >= 0
}

func CompareHealth(a, b Health) int {
	return a.CompareTo(b)
}

func (h Health) CompareTo(other Health) int {
	currentIndex := 0
	newIndex := 0
	for i, code := range healthOrder {
		if h == code {
			currentIndex = i
		}
		if other == code {
			newIndex = i
		}
	}
	if newIndex == currentIndex {
		return 0
	}
	if currentIndex > newIndex {
		return 1
	}
	return -1
}

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
	HealthStatusCrashed          HealthStatusCode = "Crashed"
	HealthStatusCreating         HealthStatusCode = "Creating"
	HealthStatusDeleted          HealthStatusCode = "Deleted"
	HealthStatusDeleting         HealthStatusCode = "Deleting"
	HealthStatusTerminating      HealthStatusCode = "Terminating"
	HealthStatusError            HealthStatusCode = "Error"
	HealthStatusRolloutFailed    HealthStatusCode = "Rollout Failed"
	HealthStatusInaccesible      HealthStatusCode = "Inaccessible"
	HealthStatusInfo             HealthStatusCode = "Info"
	HealthStatusPending          HealthStatusCode = "Pending"
	HealthStatusMaintenance      HealthStatusCode = "Maintenance"
	HealthStatusScaling          HealthStatusCode = "Scaling"
	HealthStatusRestart          HealthStatusCode = "Restarting"
	HealthStatusStarting         HealthStatusCode = "Starting"
	HealthStatusFailed           HealthStatusCode = "Failed"
	HealthStatusFailedCreate     HealthStatusCode = "Failed Create"
	HealthStatusUnschedulable    HealthStatusCode = "Unschedulable"
	HealthStatusUpgradeFailed    HealthStatusCode = "UpgradeFailed"
	HealthStatusOOMKilled        HealthStatusCode = "OOMKilled"
	HealthStatusScalingUp        HealthStatusCode = "Scaling Up"
	HealthStatusScaledToZero     HealthStatusCode = "Scaled to Zero"
	HealthStatusScalingDown      HealthStatusCode = "Scaling Down"
	HealthStatusRunning          HealthStatusCode = "Running"
	HealthStatusRollingOut       HealthStatusCode = "Rolling Out"
	HealthStatusUnhealthy        HealthStatusCode = "Unhealthy"
	HealthStatusUpdating         HealthStatusCode = "Updating"
	HealthStatusWarning          HealthStatusCode = "Warning"
	HealthStatusStopped          HealthStatusCode = "Stopped"
	HealthStatusStopping         HealthStatusCode = "Stopping"
)

// Implements custom health assessment that overrides built-in assessment
type HealthOverride interface {
	GetResourceHealth(obj *unstructured.Unstructured) (*HealthStatus, error)
}

func get(obj map[string]any, keys ...string) string {
	v, _, _ := unstructured.NestedString(obj, keys...)
	return strings.TrimSpace(v)
}

func isArgoHealth(s HealthStatusCode) bool {
	return s == "Suspended" || s == "Degraded" || s == "Progressing"
}

func GetHealthByConfigType(configType string, obj map[string]any, states ...string) HealthStatus {
	configClass := strings.Split(configType, "::")[0]

	switch strings.ToLower(configClass) {
	case "aws":
		return getAWSHealthByConfigType(configType, obj, states...)
	case "mongo":
		return GetMongoHealth(obj)
	case "kubernetes", "crossplane", "missioncontrol", "flux", "argo":
		hr, err := GetResourceHealth(&unstructured.Unstructured{Object: obj}, DefaultOverrides)
		if hr != nil {
			return *hr
		}
		if err != nil {
			return HealthStatus{
				Status:  "HealthParseError",
				Message: lo.Elipse(err.Error(), 500),
			}
		}
	}

	if len(states) > 0 {
		return GetHealthFromStatusName(states[0])
	} else {
		for k, v := range obj {
			_k := strings.ToLower(k)
			_v := fmt.Sprintf("%s", v)
			if _k == "status" || _k == "state" ||
				strings.HasSuffix(_k, "status") {
				return GetHealthFromStatusName(_v)
			}
		}
	}
	return HealthStatus{
		Health: HealthUnknown,
	}
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func GetLastUpdatedTime(obj *unstructured.Unstructured) *time.Time {
	lastUpdated := obj.GetCreationTimestamp().Time

	if obj.GetDeletionTimestamp() != nil {
		lastUpdated = maxTime(lastUpdated, obj.GetDeletionTimestamp().Time)
	}

	// Check annotations
	if annotations := obj.GetAnnotations(); annotations != nil {
		if lastApplied, ok := annotations["kubectl.kubernetes.io/last-applied-configuration"]; ok {
			if t, err := time.Parse(time.RFC3339, lastApplied); err == nil {
				lastUpdated = maxTime(lastUpdated, t)
			}
		}
	}

	possibleStatus := [][]string{
		{"lastUpdateTime"},
		{"startTime"},
		{"lastSyncTime"},
		{"reconciledAt"},
		{"startedAt"},
		{"deployedAt"},
		{"finishedAt"},
		{"lastTransitionTime"},
		{"observedAt"},
		{"operationState", "startedAt"},
		{"operationState", "finishedAt"},
	}

	// Check managed fields
	if managedFields, ok, _ := unstructured.NestedSlice(obj.Object, "metadata", "managedFields"); ok {
		for _, f := range managedFields {
			field := f.(map[string]any)
			operation, _, _ := unstructured.NestedString(field, "operation")
			if operation == "Update" {
				if updatedTime, ok := field["time"]; ok {
					if t, err := time.Parse(time.RFC3339, updatedTime.(string)); err == nil {
						lastUpdated = maxTime(lastUpdated, t)
					}
				}
			}
		}
	}

	if status, ok := obj.Object["status"].(map[string]interface{}); ok {
		for _, key := range possibleStatus {
			if value, ok, _ := unstructured.NestedString(status, key...); ok {
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					lastUpdated = maxTime(lastUpdated, t)
				}
			}
		}

		// Check conditions
		if conditions, found, _ := unstructured.NestedSlice(status, "conditions"); found {
			for _, c := range conditions {
				condition, ok := c.(map[string]interface{})
				if !ok {
					continue
				}

				for _, k := range []string{"lastProbeTime", "lastTransitionTime", "lastUpdateTime"} {
					if lastTransitionTime, exists, _ := unstructured.NestedString(condition, k); exists {
						if t, err := time.Parse(time.RFC3339, lastTransitionTime); err == nil {
							lastUpdated = maxTime(lastUpdated, t)
						}
					}
				}
			}
		}

		if containerStatuses, ok, _ := unstructured.NestedSlice(status, "containerStatuses"); ok {
			for _, c := range containerStatuses {
				containerStatus, ok := c.(map[string]interface{})
				if !ok {
					continue
				}

				for _, k := range [][]string{
					{"state", "running", "startedAt"},
					{"state", "running", "finishedAt"},
					{"state", "terminated", "finishedAt"},
					{"lastState", "terminated", "finishedAt"},
					{"operationState", "startedAt"},
					{"operationState", "finishedAt"},
				} {
					if at, exists, _ := unstructured.NestedString(containerStatus, k...); exists {
						if t, err := time.Parse(time.RFC3339, at); err == nil {
							lastUpdated = maxTime(lastUpdated, t)
						}
					}
				}
			}
		}
	}

	return &lastUpdated
}

// GetResourceHealth returns the health of a k8s resource
func GetResourceHealth(
	obj *unstructured.Unstructured,
	healthOverride HealthOverride,
) (health *HealthStatus, err error) {
	if obj.GetDeletionTimestamp() != nil && !obj.GetDeletionTimestamp().IsZero() &&
		time.Since(obj.GetDeletionTimestamp().Time) > time.Hour {
		terminatingFor := time.Since(obj.GetDeletionTimestamp().Time)
		return &HealthStatus{
			Status:      "TerminatingStalled",
			LastUpdated: GetLastUpdatedTime(obj),
			Health:      HealthWarning,
			Message: fmt.Sprintf(
				"terminating for %v",
				duration.ShortHumanDuration(terminatingFor.Truncate(time.Hour)),
			),
		}, nil
	}

	if healthCheck := GetHealthCheckFunc(obj.GroupVersionKind()); healthCheck != nil {
		if health, err = healthCheck(obj); err != nil {
			health = &HealthStatus{
				Status:  HealthStatusUnknown,
				Message: err.Error(),
			}
		}
	}

	if health == nil && healthOverride != nil {
		health, err = healthOverride.GetResourceHealth(obj)
		if err != nil {
			return &HealthStatus{
				Status:  HealthStatusUnknown,
				Message: err.Error(),
			}, err
		}
	}

	if health == nil ||
		health.Status == "" ||
		isArgoHealth(health.Status) {
		// try and get a better status from conditions
		defaultHealth, err := GetDefaultHealth(obj)
		if err != nil {
			return &HealthStatus{
				Status:  "HealthParseError",
				Message: lo.Elipse(err.Error(), 500),
			}, nil
		}
		if health == nil {
			health = defaultHealth
		}
		if health.Status == "" {
			health.Status = defaultHealth.Status
		}

		if defaultHealth.Status != "" && isArgoHealth(health.Status) && !isArgoHealth(defaultHealth.Status) {
			health.Status = defaultHealth.Status
		}
		if health.Message == "" {
			health.Message = defaultHealth.Message
		}
	}

	if health == nil {
		health = &HealthStatus{
			Status: HealthStatusUnknown,
			Ready:  true,
		}
	}
	if obj.GetDeletionTimestamp() != nil {
		health.Status = HealthStatusTerminating
		health.Ready = false
	}
	health.LastUpdated = GetLastUpdatedTime(obj)

	return health, err
}

// GetHealthCheckFunc returns built-in health check function or nil if health check is not supported
func GetHealthCheckFunc(gvk schema.GroupVersionKind) func(obj *unstructured.Unstructured) (*HealthStatus, error) {
	if gvk.Kind == "Node" {
		return getNodeHealth
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
	case "canaries.flanksource.com":
		switch gvk.Kind {
		case "Canary":
			return getCanaryHealth
			// case "Component":
			// case "Topology":
		}
	case "configs.flanksource.com":
		switch gvk.Kind {
		case "ScrapeConfig":
			return getScrapeConfigHealth
			// 	case "ScrapePlugin":
		}
	case "mission-control.flanksource.com":
		switch gvk.Kind {
		case "Notification":
			return getNotificationHealth
			// 	case "Playbook":
			// 	case "NotificationSilence":
			// 	case "Connection":
		}
	case "kustomize.toolkit.fluxcd.io", "helm.toolkit.fluxcd.io", "source.toolkit.fluxcd.io":
		return GetDefaultHealth
	case "cert-manager.io":
		switch gvk.Kind {
		case "CertificateRequest":
			return GetCertificateRequestHealth
		default:
			return GetCertificateHealth
		}
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

func init() {
	properties.RegisterListener(func(p *properties.Properties) {
		if v := p.Duration(defaultCertExpiryWarningPeriod, "health.cert-manager.expiryGracePeriod"); v != 0 {
			certExpiryWarningPeriod = v
		}

		if v := p.Duration(defaultCrtRenewalWarningPeriod, "health.cert-manager.renewalGracePeriod"); v != 0 {
			certRenewalWarningPeriod = v
		}
	})
}
