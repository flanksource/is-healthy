package health

import (
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getPodHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case corev1.SchemeGroupVersion.WithKind(PodKind):
		var pod corev1.Pod
		err := convertFromUnstructured(obj, &pod)
		if err != nil {
			return nil, err
		}
		return getCorev1PodHealth(&pod)
	default:
		return nil, fmt.Errorf("unsupported Pod GVK: %s", gvk)
	}
}

func getPodStatus(containers ...corev1.ContainerStatus) (waiting *HealthStatus, terminated *HealthStatus) {
	for _, container := range containers {
		_waiting, _terminated := getContainerStatus(container)
		if _waiting != nil {
			if waiting == nil {
				waiting = _waiting
			} else if _waiting.Health.IsWorseThan(waiting.Health) {
				waiting = _waiting
			}
		}
		if _terminated != nil {
			if terminated == nil {
				terminated = _terminated
			} else if _terminated.Health.IsWorseThan(terminated.Health) {
				terminated = _terminated
			}
		}
	}
	return waiting, terminated
}

func isErrorStatus(s string) bool {
	return strings.HasPrefix(s, "Err") ||
		strings.HasSuffix(s, "Error") ||
		strings.HasSuffix(s, "BackOff")
}

func getContainerStatus(containerStatus corev1.ContainerStatus) (waiting *HealthStatus, terminated *HealthStatus) {
	if state := containerStatus.State.Waiting; state != nil {
		waiting = &HealthStatus{
			Status: HealthStatusCode(state.Reason),
			Health: lo.Ternary(
				isErrorStatus(state.Reason) || containerStatus.RestartCount > 0,
				HealthUnhealthy,
				HealthUnknown,
			),
			Message: state.Message,
		}
	}

	if state := containerStatus.LastTerminationState.Terminated; state != nil {
		age := time.Since(state.FinishedAt.Time)
		// ignore old terminate statuses
		if age < time.Hour*24 {
			terminated = &HealthStatus{
				Status:  HealthStatusCode(state.Reason),
				Health:  lo.Ternary(age < time.Hour, HealthUnhealthy, HealthWarning),
				Message: state.Message,
			}
			if state.Reason == string(HealthStatusCompleted) && state.ExitCode == 0 {
				// completed successfully
				terminated.Health = HealthHealthy
			}
		}
	}
	return waiting, terminated
}

func getCorev1PodHealth(pod *corev1.Pod) (*HealthStatus, error) {
	isReady := IsPodReady(pod)
	containers := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
	deadline := GetStartDeadline(append(pod.Spec.InitContainers, pod.Spec.Containers...)...)
	age := time.Since(pod.CreationTimestamp.Time).Truncate(time.Minute).Abs()
	isStarting := age < deadline
	hr := HealthStatus{
		Health: lo.Ternary(isReady, HealthHealthy, HealthUnhealthy),
	}

	if pod.ObjectMeta.DeletionTimestamp != nil && !pod.ObjectMeta.DeletionTimestamp.IsZero() {
		status := HealthUnknown
		message := ""

		terminatingFor := time.Since(pod.ObjectMeta.DeletionTimestamp.Time)
		if terminatingFor >= time.Minute*15 {
			status = HealthWarning
			message = fmt.Sprintf("stuck in 'Terminating' for %s", terminatingFor.Truncate(time.Minute))
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

	for _, ctrStatus := range pod.Status.Conditions {
		if ctrStatus.Reason == "Unschedulable" {
			return &HealthStatus{
				Health:  HealthUnhealthy,
				Status:  HealthStatusUnschedulable,
				Message: ctrStatus.Message,
			}, nil
		}
	}

	waiting, terminated := getPodStatus(containers...)

	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		return &HealthStatus{
			Health:  HealthHealthy,
			Status:  HealthStatusCompleted,
			Ready:   true,
			Message: pod.Status.Message,
		}, nil

	case corev1.PodFailed:
		hr.Health = HealthUnhealthy
		hr.Ready = true
		hr.Status, _ = lo.Coalesce(hr.Status, HealthStatusFailed)
		hr.Message = lo.CoalesceOrEmpty(pod.Status.Message, hr.Message)

	case corev1.PodRunning, corev1.PodPending:
		hr = hr.Merge(terminated, waiting)
		if terminated != nil && terminated.Health.IsWorseThan(HealthWarning) &&
			hr.Status == HealthStatusCrashLoopBackoff {
			hr.Status = terminated.Status
			hr.Health = hr.Health.Worst(terminated.Health)
		}
		hr.Status, _ = lo.Coalesce(hr.Status, HealthStatusRunning)
		hr.Health = hr.Health.Worst(lo.Ternary(isReady, HealthHealthy, HealthUnhealthy))
	}

	if isStarting && hr.Health.IsWorseThan(HealthWarning) &&
		(terminated != nil && terminated.Status != HealthStatusOOMKilled) {
		hr.Health = HealthUnknown
		hr.Message = fmt.Sprintf("%s %s", string(hr.Status), hr.Message)
		hr.Status = HealthStatusStarting
	}

	return &hr, nil
}

type ContainerRecord struct {
	Spec   corev1.Container
	Status corev1.ContainerStatus
}
