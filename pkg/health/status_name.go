package health

import (
	"strings"

	"github.com/samber/lo"
)

func GetHealthFromStatusName(status string) (health HealthStatus) {
	if status == "" {
		return HealthStatus{}
	}

	status = strings.ReplaceAll(status, "_", " ")
	status = strings.ReplaceAll(status, "-", " ")

	words := lo.Words(status)
	for i, word := range words {
		words[i] = lo.Capitalize(word)
	}
	hr := HealthStatus{
		Status: HealthStatusCode(strings.Join(words, " ")),
	}

	status = strings.ToLower(status)

	switch status {
	case "update complete cleanup in progress",
		"update in progress",
		"updating",
		"maintenance",
		"rebooting",
		"storage full",
		"storage optimization",
		"upgrading",
		"resetting master credentials",
		"modifying":
		hr.Health = HealthHealthy
	case "stopped", "terminated", "delete complete", "deleted":
		hr.Health = HealthUnknown
		hr.Ready = true
	case "stopping", "shutting down", "delete in progress", "import in progress", "deleting":
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
	case "running", "active", "create complete", "import complete", "update complete", "available", "in use":
		hr.Health = HealthHealthy
		hr.Ready = true
	case "rollback in progress", "import rollback in progress", "update rollback in progress":
		hr.Health = HealthWarning
	case "import rollback complete", "rollback complete", "update rollback complete", "active impaired":
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

	return hr
}
