package health

import (
	"strings"

	"github.com/samber/lo"
)

func GetECSTaskHealth(obj map[string]any) (health HealthStatus) {
	hr := HealthStatus{
		Status: HealthStatusCode(lo.CamelCase(obj["LastStatus"].(string))),
		Health: HealthUnknown,
		Ready:  false,
	}

	if v, ok := obj["HealthStatus"].(string); ok {
		hr.Health = Health(lo.CamelCase(v))
	}

	switch hr.Status {
	case "RUNNING":
		hr.Health = HealthHealthy
		hr.Ready = true
	case "STOPPED", "DELETED":
		hr.Ready = true
		hr.Health = HealthUnknown
	}

	stopCode, _ := obj["StopCode"].(string)

	if stopCode != "" {
		hr.Status = HealthStatusCode(stopCode)
	}
	switch stopCode {
	case "TaskFailedToStart":
		hr.Health = HealthUnhealthy
	case "EssentialContainerExited":
		hr.Status = HealthStatusCrashed
		hr.Health = HealthUnhealthy
	case "UserInitiated":
		hr.Status = HealthStatusStopped
	case "ServiceSchedulerInitiated":
		hr.Status = HealthStatusTerminating
	}

	if reason, ok := obj["StoppedReason"].(string); ok {
		idx := strings.Index(reason, ":")

		if idx > 0 {
			hr.Status = HealthStatusCode(reason[0:idx])
			if len(reason) >= idx+1 {
				hr.Message = strings.TrimSpace(reason[idx+1:])
			}

			switch hr.Status {
			case "ContainerRuntimeError", "ContainerRuntimeTimeoutError", "OutOfMemoryError":
				hr.Health = HealthUnhealthy
			case "InternalError", "CannotCreateVolumeError", "ResourceNotFoundException", "CannotStartContainerError":
				hr.Health = HealthUnhealthy
				hr.Ready = true
			case "SpotInterruptionError", "CannotStopContainerError", "CannotInspectContainerError":
				hr.Health = HealthWarning
			case "TaskFailedToStart", "ResourceInitializationError", "CannotPullContainer":
				hr.Health = HealthUnhealthy
			default:
				hr.Health = HealthUnhealthy
			}
		}
	}

	return hr
}
