package health

import (
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getPodHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case corev1.SchemeGroupVersion.WithKind(PodKind):
		var pod corev1.Pod
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &pod)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured Pod to typed: %v", err)
		}
		return getCorev1PodHealth(&pod)
	default:
		return nil, fmt.Errorf("unsupported Pod GVK: %s", gvk)
	}
}

func getCorev1PodHealth(pod *corev1.Pod) (*HealthStatus, error) {
	isReady := IsPodReady(pod)
	if pod.ObjectMeta.DeletionTimestamp != nil && !pod.ObjectMeta.DeletionTimestamp.IsZero() {
		status := HealthUnknown
		message := ""

		terminatingFor := time.Since(pod.ObjectMeta.DeletionTimestamp.Time)
		if terminatingFor >= time.Minute*15 {
			status = HealthWarning
			message = fmt.Sprintf("stuck in 'Terminating' for %s", terminatingFor)
		}

		return &HealthStatus{
			Status:  HealthStatusTerminating,
			Ready:   false,
			Health:  status,
			Message: message,
		}, nil
	}

	if pod.Status.Reason == "Evicted" {
		return &HealthStatus{
			Health:  HealthWarning,
			Status:  HealthStatusEvicted,
			Ready:   true,
			Message: pod.Status.Message,
		}, nil
	}

	getCommonContainerError := func(containerStatus *corev1.ContainerStatus) *HealthStatus {
		waiting := containerStatus.State.Waiting
		// Article listing common container errors: https://medium.com/kokster/debugging-crashloopbackoffs-with-init-containers-26f79e9fb5bf
		if waiting != nil && (strings.HasPrefix(waiting.Reason, "Err") || strings.HasSuffix(waiting.Reason, "Error") || strings.HasSuffix(waiting.Reason, "BackOff")) {
			return &HealthStatus{
				Status:  HealthStatusCode(waiting.Reason),
				Health:  HealthUnhealthy,
				Message: waiting.Message,
			}
		}

		return nil
	}

	// This logic cannot be applied when the pod.Spec.RestartPolicy is: corev1.RestartPolicyOnFailure,
	// corev1.RestartPolicyNever, otherwise it breaks the resource hook logic.
	// The issue is, if we mark a pod with ImagePullBackOff as Degraded, and the pod is used as a resource hook,
	// then we will prematurely fail the PreSync/PostSync hook. Meanwhile, when that error condition is resolved
	// (e.g. the image is available), the resource hook pod will unexpectedly be executed even though the sync has
	// completed.
	if pod.Spec.RestartPolicy == corev1.RestartPolicyAlways {
		var status HealthStatusCode
		var health Health
		var messages []string

		for _, containerStatus := range pod.Status.ContainerStatuses {
			if msg := getCommonContainerError(&containerStatus); msg != nil {
				health = msg.Health
				status = msg.Status
				messages = append(messages, msg.Message)
			}
		}

		if status != "" {
			return &HealthStatus{
				Health:  health,
				Status:  status,
				Message: strings.Join(messages, ", "),
			}, nil
		}
	}

	getFailMessage := func(ctr *corev1.ContainerStatus) string {
		if ctr.State.Terminated != nil {
			if ctr.State.Terminated.Message != "" {
				return ctr.State.Terminated.Message
			}
			if ctr.State.Terminated.Reason == "OOMKilled" {
				return ctr.State.Terminated.Reason
			}
			if ctr.State.Terminated.ExitCode != 0 {
				return fmt.Sprintf("container %q failed with exit code %d", ctr.Name, ctr.State.Terminated.ExitCode)
			}
		}
		return ""
	}

	switch pod.Status.Phase {
	case corev1.PodPending:
		for _, ctrStatus := range pod.Status.InitContainerStatuses {
			if ctrStatus.LastTerminationState.Terminated != nil && ctrStatus.LastTerminationState.Terminated.Reason == "Error" {
				// A pending pod whose container was previously terminated with error should be marked as unhealthy (instead of unknown)
				return &HealthStatus{
					Health:  HealthUnhealthy,
					Status:  HealthStatusCrashLoopBackoff,
					Message: ctrStatus.LastTerminationState.Terminated.Reason,
				}, nil
			}

			if msg := getCommonContainerError(&ctrStatus); msg != nil {
				return msg, nil
			}
		}

		for _, ctrStatus := range pod.Status.Conditions {
			if ctrStatus.Reason == "Unschedulable" {
				return &HealthStatus{
					Health:  HealthUnhealthy,
					Status:  HealthStatusUnschedulable,
					Message: ctrStatus.Message,
				}, nil
			}
		}

		return &HealthStatus{
			Health:  HealthUnknown,
			Status:  HealthStatusPending,
			Message: pod.Status.Message,
		}, nil

	case corev1.PodSucceeded:
		return &HealthStatus{
			Health:  HealthHealthy,
			Status:  HealthStatusCompleted,
			Ready:   true,
			Message: pod.Status.Message,
		}, nil

	case corev1.PodFailed:
		if pod.Status.Message != "" {
			// Pod has a nice error message. Use that.
			return &HealthStatus{Health: HealthUnhealthy, Status: HealthStatusError, Ready: true, Message: pod.Status.Message}, nil
		}
		for _, ctr := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
			if msg := getFailMessage(&ctr); msg != "" {
				return &HealthStatus{Health: HealthUnhealthy, Status: HealthStatusError, Ready: true, Message: msg}, nil
			}
		}

		return &HealthStatus{Health: HealthUnhealthy, Status: HealthStatusError, Message: "", Ready: true}, nil

	case corev1.PodRunning:
		switch pod.Spec.RestartPolicy {
		case corev1.RestartPolicyAlways:
			if isReady {
				h := &HealthStatus{
					Health:  HealthHealthy,
					Ready:   true,
					Status:  HealthStatusRunning,
					Message: pod.Status.Message,
				}

				// A ready pod can be in a warning state if it has been in a restart loop.
				// i.e. the container completes successfully, but the pod keeps restarting.
				for _, s := range pod.Status.ContainerStatuses {
					possiblyInRestartLoop := s.RestartCount > 2 &&
						s.LastTerminationState.Terminated != nil &&
						time.Since(s.State.Running.StartedAt.Time) < time.Hour*4

					if possiblyInRestartLoop {
						lastTerminatedTime := s.LastTerminationState.Terminated.FinishedAt.Time
						h.Message = fmt.Sprintf("%s has restarted %d time(s)", s.Name, pod.Status.ContainerStatuses[0].RestartCount)

						if s.LastTerminationState.Terminated.Reason != "Completed" {
							h.Status = HealthStatusCode(s.LastTerminationState.Terminated.Reason)
						}

						if time.Since(lastTerminatedTime) < time.Minute*30 {
							h.Health = HealthUnhealthy
							h.Ready = false
						} else if time.Since(lastTerminatedTime) < time.Hour*8 {
							h.Health = HealthWarning
							h.Ready = false
						}
					}
				}

				return h, nil
			}

			// if it's not ready, check to see if any container terminated, if so, it's degraded
			var nonReadyContainers []ContainerRecord
			for _, ctrStatus := range pod.Status.ContainerStatuses {
				if !ctrStatus.Ready {
					spec := lo.Filter(pod.Spec.Containers, func(i corev1.Container, _ int) bool {
						return i.Name == ctrStatus.Name
					})
					nonReadyContainers = append(nonReadyContainers, ContainerRecord{
						Status: ctrStatus,
						Spec:   spec[0],
					})
				}

				if ctrStatus.LastTerminationState.Terminated != nil {
					return &HealthStatus{
						Health:  HealthUnhealthy,
						Ready:   true,
						Status:  HealthStatusCode(ctrStatus.LastTerminationState.Terminated.Reason),
						Message: ctrStatus.LastTerminationState.Terminated.Message,
					}, nil
				}
			}

			// Pod isn't ready but all containers are
			if len(nonReadyContainers) == 0 {
				return &HealthStatus{
					Health:  HealthWarning,
					Status:  HealthStatusRunning,
					Message: pod.Status.Message,
				}, nil
			}

			var containersWaitingForReadinessProbe []string
			for _, c := range nonReadyContainers {
				if c.Spec.ReadinessProbe == nil || c.Spec.ReadinessProbe.InitialDelaySeconds == 0 {
					continue
				}

				if c.Status.State.Running != nil && time.Since(c.Status.State.Running.StartedAt.Time) <= time.Duration(c.Spec.ReadinessProbe.InitialDelaySeconds)*time.Second {
					containersWaitingForReadinessProbe = append(containersWaitingForReadinessProbe, c.Spec.Name)
				}
			}

			// otherwise we are progressing towards a ready state
			return &HealthStatus{
				Health:  HealthUnknown,
				Status:  HealthStatusStarting,
				Message: fmt.Sprintf("Container %s is waiting for readiness probe", strings.Join(containersWaitingForReadinessProbe, ",")),
			}, nil

		case corev1.RestartPolicyOnFailure, corev1.RestartPolicyNever:
			if isReady {
				return &HealthStatus{
					Health: HealthHealthy,
					Status: HealthStatusRunning,
				}, nil
			} else {
				return &HealthStatus{
					Health: HealthUnhealthy,
					Status: HealthStatusRunning,
				}, nil
			}
		}
	}

	return &HealthStatus{
		Health:  HealthUnknown,
		Status:  HealthStatusUnknown,
		Message: pod.Status.Message,
	}, nil
}

type ContainerRecord struct {
	Spec   corev1.Container
	Status corev1.ContainerStatus
}
