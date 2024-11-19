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
		hr.Status = HealthStatusCode(HumanCase(v))
	}

	switch strings.ToUpper(string(hr.Status)) {
	case "RUNNING":
		hr.Health = HealthHealthy
		hr.Ready = true
	case "STOPPED", "DELETED", "UNKNOWN":
		hr.Ready = true
		hr.Health = HealthUnknown
	}

	stopCode, _ := obj["StopCode"].(string)

	if stopCode != "" {
		hr.Status = HealthStatusCode(stopCode)
	}

	switch strings.ToUpper(stopCode) {
	case "TASKFAILEDTOSTART":
		hr.Health = HealthUnhealthy
	case "ESSENTIALCONTAINEREXITED":
		hr.Status = HealthStatusCrashed
		hr.Health = HealthUnhealthy
	case "USERINITIATED":
		hr.Status = HealthStatusStopped
	case "SERVICESCHEDULERINITIATED":
		hr.Status = HealthStatusTerminating
	}

	if reason, ok := obj["StoppedReason"].(string); ok {
		idx := strings.Index(reason, ":")

		if idx > 0 {
			hr.Status = HealthStatusCode(reason[0:idx])
			if len(reason) >= idx+1 {
				hr.Message = strings.TrimSpace(reason[idx+1:])
			}

			switch strings.ToUpper(string(hr.Status)) {
			case "CONTAINERRUNTIMEERROR", "CONTAINERRUNTIMETIMEOUTERROR", "OUTOFMEMORYERROR":
				hr.Health = HealthUnhealthy
			case "INTERNALERROR", "CANNOTCREATEVOLUMEERROR", "RESOURCENOTFOUNDERROR", "CANNOTSTARTCONTAINERERROR":
				hr.Health = HealthUnhealthy
				hr.Ready = true
			case "SPOTINTERRUPTIONERROR", "CANNOTSTOPCONTAINERERROR", "CANNOTINSPECTCONTAINERERROR":
				hr.Health = HealthWarning
			case "TASKFAILEDTOSTART", "RESOURCEINITIALIZATIONERROR", "CANNOTPULLCONTAINER":
				hr.Health = HealthUnhealthy
			default:
				hr.Health = HealthUnhealthy
			}
		}
	}

	return hr
}
