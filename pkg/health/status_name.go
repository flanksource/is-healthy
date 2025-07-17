package health

import (
	"strings"
)

func GetHealthFromStatusName(status string, reasons ...string) (health HealthStatus) {
	if status == "" {
		return HealthStatus{}
	}

	hr := HealthStatus{
		Status: HealthStatusCode(HumanCase(status)),
	}

	switch strings.ToLower(string(hr.Status)) {
	case "update complete cleanup in progress",
		"update in progress",
		"updating",
		"rebooting",
		"storage full",
		"storage optimization",
		"upgrading",
		"resetting master credentials",
		"reconciling",
		"modifying":
		hr.Health = HealthHealthy
	case "stopped", "terminated", "delete complete", "deleted":
		hr.Health = HealthUnknown
		hr.Ready = true
	case "creating", "stopping", "shutting down", "delete in progress", "repairing",
		"import in progress", "deleting", "provisioning", "staging", "suspending":
		hr.Health = HealthUnknown
	case "create failed",
		"delete failed",
		"import rollback failed",
		"rollback failed",
		"update failed",
		"update rollback failed",
		"failed",
		"error",
		"insufficient capacity":
		hr.Health = HealthUnhealthy
		hr.Ready = true
	case "running", "active", "create complete", "ready",
		"import complete", "update complete", "available", "in use":
		hr.Health = HealthHealthy
		hr.Ready = true
	case "rollback in progress",
		"import rollback in progress",
		"update rollback in progress",
		"maintenance", "degraded",
		"restoring":
		hr.Health = HealthWarning
	case "suspended", "import rollback complete", "rollback complete", "running_with_error",
		"update rollback complete", "active impaired":
		hr.Health = HealthWarning
		hr.Ready = true
	}

	if hr.Health == "" {
		switch {
		case strings.HasPrefix(status, "inaccessible") || strings.HasPrefix(status, "incompatible") || strings.Contains(status, "error"):
			hr.Health = HealthUnhealthy
			hr.Ready = true
		case strings.HasPrefix(status, "configuring"):
			hr.Health = HealthHealthy
		}
	}

	for _, v := range reasons {
		if v != "" {
			hr.Message = v
			break
		}
	}

	return hr
}
