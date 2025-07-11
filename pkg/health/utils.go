package health

import (
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/util/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	SecretKind                   = "Secret"
	ServiceKind                  = "Service"
	ServiceAccountKind           = "ServiceAccount"
	EndpointsKind                = "Endpoints"
	DeploymentKind               = "Deployment"
	ReplicaSetKind               = "ReplicaSet"
	StatefulSetKind              = "StatefulSet"
	DaemonSetKind                = "DaemonSet"
	IngressKind                  = "Ingress"
	JobKind                      = "Job"
	CronJobKind                  = "CronJob"
	PersistentVolumeClaimKind    = "PersistentVolumeClaim"
	CustomResourceDefinitionKind = "CustomResourceDefinition"
	PodKind                      = "Pod"
	APIServiceKind               = "APIService"
	NamespaceKind                = "Namespace"
	HorizontalPodAutoscalerKind  = "HorizontalPodAutoscaler"
)

type HealthStatus struct {
	// Ready indicates the resource has reached its desired state and is not actively changing.
	// A resource is considered ready when it has successfully completed its deployment/update
	// and is stable (not progressing through state transitions).
	Ready  bool   `json:"ready"`
	Health Health `json:"health"`
	// Status holds the status code of the application or resource
	Status HealthStatusCode `json:"status,omitempty"      protobuf:"bytes,1,opt,name=status"`
	// Message is a human-readable informational message describing the health status
	Message string `json:"message,omitempty"     protobuf:"bytes,2,opt,name=message"`
	// LastUpdated is the time this resource as last updated, detected by inspecting all
	// of the relevant status timestamps
	LastUpdated *time.Time `json:"lastUpdated,omitempty"`

	order int `json:"-" yaml:"-"`
}

func (hs HealthStatus) String() string {
	m := string(hs.Status)

	if hs.Health != HealthUnknown {
		if m != "" {
			m += " "
		}
		m += fmt.Sprintf("(%s)", hs.Health)
	}

	if len(hs.Message) > 0 {
		if m != "" {
			m += ": "
		}
		m += hs.Message
	}
	return m
}

func (hs HealthStatus) Merge(others ...*HealthStatus) HealthStatus {
	for _, other := range others {
		if other == nil {
			continue
		}
		hs.Status = HealthStatusCode(lo.CoalesceOrEmpty(string(hs.Status), string(other.Status)))
		hs.Health = hs.Health.Worst(other.Health)
		if other.Health.IsWorseThan(hs.Health) {
			hs.Health = other.Health
			hs.Status = other.Status
		}
		hs.Ready = hs.Ready && other.Ready
		hs.Message = strings.TrimSpace(hs.Message)
		if hs.Message != "" {
			hs.Message = strings.Join(lo.Compact([]string{hs.Message, other.Message}), ", ")
		} else {
			hs.Message = other.Message
		}
	}
	return hs
}

func (hs *HealthStatus) AppendMessage(msg string, args ...interface{}) {
	if msg == "" {
		return
	}
	if strings.TrimSpace(hs.Message) != "" {
		hs.Message += ", "
	}
	hs.Message += fmt.Sprintf(msg, args...)
}

func (hs *HealthStatus) PrependMessage(msg string, args ...interface{}) {
	if msg == "" {
		return
	}
	if hs.Message != "" {
		hs.Message = fmt.Sprintf(msg, args...) + ", " + hs.Message
	} else {
		hs.Message = fmt.Sprintf(msg, args...)
	}
}

// IsPodAvailable returns true if a pod is available; false otherwise.
// Precondition for an available pod is that it must be ready. On top
// of that, there are two cases when a pod can be considered available:
// 1. minReadySeconds == 0, or
// 2. LastTransitionTime (is set) + minReadySeconds < current time
func IsPodAvailable(pod *corev1.Pod, minReadySeconds int32, now metav1.Time) bool {
	if isReady, _ := IsPodReady(pod); !isReady {
		return false
	}

	c := getPodReadyCondition(pod.Status)
	minReadySecondsDuration := time.Duration(minReadySeconds) * time.Second
	if minReadySeconds == 0 ||
		!c.LastTransitionTime.IsZero() && c.LastTransitionTime.Add(minReadySecondsDuration).Before(now.Time) {
		return true
	}
	return false
}

// IsPodReady returns true if a pod is ready; false with message otherwise.
func IsPodReady(pod *corev1.Pod) (bool, string) {
	return isPodReadyConditionTrue(pod.Status)
}

// IsPodReadyConditionTrue returns true if a pod is ready; false otherwise.
func isPodReadyConditionTrue(status corev1.PodStatus) (bool, string) {
	condition := getPodReadyCondition(status)
	if condition == nil {
		return false, ""
	}
	msg := fmt.Sprintf("%s %s", condition.Reason, condition.Message)
	return condition.Status == corev1.ConditionTrue, msg
}

// GetPodReadyCondition extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func getPodReadyCondition(status corev1.PodStatus) *corev1.PodCondition {
	_, condition := getPodCondition(&status, corev1.PodReady)
	return condition
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func getPodCondition(status *corev1.PodStatus, conditionType corev1.PodConditionType) (int, *corev1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	return getPodConditionFromList(status.Conditions, conditionType)
}

// GetPodConditionFromList extracts the provided condition from the given list of condition and
// returns the index of the condition and the condition. Returns -1 and nil if the condition is not present.
func getPodConditionFromList(
	conditions []corev1.PodCondition,
	conditionType corev1.PodConditionType,
) (int, *corev1.PodCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}

func convertFromUnstructured[T any](o *unstructured.Unstructured, to *T) error {
	js, err := json.Marshal(o)
	if err != nil {
		return fmt.Errorf("failed to marshal object to JSON: %w", err)
	}

	if err = json.Unmarshal(js, to); err != nil {
		return fmt.Errorf("failed to unmarshal object into: %T: %v", *to, err)
	}
	return nil
}

// duration after the creation of a replica set
// within which we never deem the it to be unhealthy.
const PodStartingBufferPeriod = time.Minute * 10

func GetStartDeadline(containers ...corev1.Container) time.Duration {
	max := PodStartingBufferPeriod
	for _, container := range containers {
		if readiness := container.ReadinessProbe; readiness != nil {
			podLevel := time.Second * time.Duration(
				readiness.InitialDelaySeconds+readiness.FailureThreshold*(readiness.PeriodSeconds+readiness.TimeoutSeconds),
			)
			if podLevel > max {
				max = podLevel
			}
		}
	}
	return max.Truncate(time.Minute)
}

func IsContainerStarting(creation time.Time, containers ...corev1.Container) bool {
	return time.Since(creation) < GetStartDeadline(containers...)
}

func HumanCase(s string) string {
	// Add space before capital letters
	for i := len(s) - 2; i >= 0; i-- {
		if s[i] >= 'a' && s[i] <= 'z' && s[i+1] >= 'A' && s[i+1] <= 'Z' {
			s = s[:i+1] + " " + s[i+1:]
		}
	}

	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	items := strings.Split(strings.TrimSpace(s), " ")
	for i := range items {
		items[i] = lo.Capitalize(items[i])
	}
	return strings.Join(items, " ")
}
