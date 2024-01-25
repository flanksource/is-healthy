package health

import "strings"

// MapAWSStatus maps an AWS resource's statuses to a Health Code
func MapAWSStatus(status string) string {
	if s, found := awsStatusMap[strings.ToLower(status)]; found {
		return string(s)
	}

	return string(HealthStatusUnknown)
}

var awsStatusMap = map[string]HealthStatusCode{
	"available":     HealthStatusHealthy,
	"pending":       HealthStatusPending,
	"running":       HealthStatusHealthy,
	"shutting-down": HealthStatusDeleting,
	"stopped":       HealthStatusSuspended,
	"stopping":      HealthStatusDeleting,
	"terminated":    HealthStatusDeleted,

	// EKS
	"creating": HealthStatusCreating,
	"active":   HealthStatusHealthy,
	"deleting": HealthStatusDeleting,
	"failed":   HealthStatusError,
	"updating": HealthStatusUpdating,
	// "pending":  HealthStatusPending,

	// EBS
	// "creating": HealthStatusCreating,
	// "available": HealthStatusUpdating,
	"in-use": HealthStatusHealthy,
	// "deleting": HealthStatusDeleting,
	"deleted": HealthStatusDeleted,
	"error":   HealthStatusError,

	// RDS Status
	// "available":                           HealthStatusHealthy,
	"billed":                          HealthStatusHealthy,
	"backing-up":                      HealthStatusMaintenance,
	"configuring-enhanced-monitoring": HealthStatusMaintenance,
	"configuring-iam-database-auth":   HealthStatusMaintenance,
	"configuring-log-exports":         HealthStatusMaintenance,
	"converting-to-vpc":               HealthStatusUpdating,
	// "creating":                            HealthStatusCreating,
	"delete-precheck": HealthStatusMaintenance,
	// "deleting":                            HealthStatusDeleting,
	// "failed":                              HealthStatusUnhealthy,
	"inaccessible-encryption-credentials":             HealthStatusInaccesible,
	"inaccessible-encryption-credentials-recoverable": HealthStatusInaccesible,
	"incompatible-network":                            HealthStatusUnhealthy,
	"incompatible-option-group":                       HealthStatusUnhealthy,
	"incompatible-parameters":                         HealthStatusUnhealthy,
	"incompatible-restore":                            HealthStatusUnhealthy,
	"insufficient-capacity":                           HealthStatusUnhealthy,
	"maintenance":                                     HealthStatusMaintenance,
	"modifying":                                       HealthStatusUpdating,
	"moving-to-vpc":                                   HealthStatusMaintenance,
	"rebooting":                                       HealthStatusStopping,
	"resetting-master-credentials":                    HealthStatusMaintenance,
	"renaming":                                        HealthStatusMaintenance,
	"restore-error":                                   HealthStatusError,
	"starting":                                        HealthStatusProgressing,
	// "stopped":                                         HealthStatusSuspended,
	// "stopping":                                        HealthStatusStopping,
	"storage-config-upgrade": HealthStatusUpdating,
	"storage-full":           HealthStatusUnhealthy,
	"storage-optimization":   HealthStatusMaintenance,
	"upgrading":              HealthStatusUpdating,

	// ELB
	// "active":          HealthStatusHealthy,
	"provisioning":    HealthStatusProgressing,
	"active_impaired": HealthStatusWarning,
	// "failed":          HealthStatusError,
}
