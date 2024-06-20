package health

import (
	"fmt"
	"strings"

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
		if isReady {
			return &HealthStatus{
				Status: HealthStatusTerminating,
				Ready:  false,
				Health: HealthHealthy,
			}, nil
		} else {
			return &HealthStatus{
				Status: HealthStatusTerminating,
				Ready:  false,
				Health: HealthUnhealthy,
			}, nil
		}
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
			if msg := getCommonContainerError(&ctrStatus); msg != nil {
				return msg, nil
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
			// if pod is ready, it is automatically healthy
			if isReady {
				return &HealthStatus{
					Health:  HealthHealthy,
					Ready:   true,
					Status:  HealthStatusRunning,
					Message: pod.Status.Message,
				}, nil
			}
			// if it's not ready, check to see if any container terminated, if so, it's degraded
			for _, ctrStatus := range pod.Status.ContainerStatuses {
				if ctrStatus.LastTerminationState.Terminated != nil {
					return &HealthStatus{
						Health:  HealthUnhealthy,
						Ready:   true,
						Status:  HealthStatusCode(ctrStatus.LastTerminationState.Terminated.Reason),
						Message: ctrStatus.LastTerminationState.Terminated.Message,
					}, nil
				}
			}
			// otherwise we are progressing towards a ready state
			return &HealthStatus{
				Health:  HealthUnknown,
				Status:  HealthStatusStarting,
				Message: pod.Status.Message,
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
