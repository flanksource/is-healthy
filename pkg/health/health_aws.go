package health

import "strings"

const (
	AWSResourceTypeEBS    string = "ebs"
	AWSResourceTypeEC2    string = "ec2"
	AWSResourceTypeEKS    string = "eks"
	AWSResourceTypeELB    string = "elb"
	AWSResourceTypeRDS    string = "rds"
	AWSResourceTypeVPC    string = "vpc"
	AWSResourceTypeSubnet string = "subnet"
)

// MapAWSStatus maps an AWS resource's statuses to a Health Code
func MapAWSStatus(status string, resourceType string) string {
	if resourceStatuses, found := awsStatusMap[resourceType]; found {
		if v, found := resourceStatuses[strings.ToLower(status)]; found {
			return string(v)
		}
	}

	return string(HealthStatusUnknown)
}

var awsStatusMap = map[string]map[string]HealthStatusCode{
	AWSResourceTypeEC2: {
		"pending":       HealthStatusPending,
		"running":       HealthStatusHealthy,
		"shutting-down": HealthStatusDeleting,
		"stopped":       HealthStatusStopped,
		"stopping":      HealthStatusStopping,
		"terminated":    HealthStatusDeleted,
	},

	AWSResourceTypeEKS: {
		"creating": HealthStatusCreating,
		"active":   HealthStatusHealthy,
		"deleting": HealthStatusDeleting,
		"failed":   HealthStatusError,
		"updating": HealthStatusUpdating,
		"pending":  HealthStatusPending,
	},

	AWSResourceTypeEBS: {
		"creating":  HealthStatusCreating,
		"available": HealthStatusStopped,
		"in-use":    HealthStatusHealthy,
		"deleting":  HealthStatusDeleting,
		"deleted":   HealthStatusDeleted,
		"error":     HealthStatusError,
	},

	AWSResourceTypeRDS: {
		"available":                           HealthStatusHealthy,
		"billed":                              HealthStatusHealthy,
		"backing-up":                          HealthStatusMaintenance,
		"configuring-enhanced-monitoring":     HealthStatusMaintenance,
		"configuring-iam-database-auth":       HealthStatusMaintenance,
		"configuring-log-exports":             HealthStatusMaintenance,
		"converting-to-vpc":                   HealthStatusUpdating,
		"creating":                            HealthStatusCreating,
		"delete-precheck":                     HealthStatusMaintenance,
		"deleting":                            HealthStatusDeleting,
		"failed":                              HealthStatusUnhealthy,
		"inaccessible-encryption-credentials": HealthStatusInaccesible,
		"inaccessible-encryption-credentials-recoverable": HealthStatusInaccesible,
		"incompatible-network":                            HealthStatusUnhealthy,
		"incompatible-option-group":                       HealthStatusUnhealthy,
		"incompatible-parameters":                         HealthStatusUnhealthy,
		"incompatible-restore":                            HealthStatusUnhealthy,
		"insufficient-capacity":                           HealthStatusUnhealthy,
		"maintenance":                                     HealthStatusMaintenance,
		"modifying":                                       HealthStatusUpdating,
		"moving-to-vpc":                                   HealthStatusMaintenance,
		"rebooting":                                       HealthStatusRestart,
		"resetting-master-credentials":                    HealthStatusMaintenance,
		"renaming":                                        HealthStatusMaintenance,
		"restore-error":                                   HealthStatusError,
		"starting":                                        HealthStatusStarting,
		"stopped":                                         HealthStatusStopped,
		"stopping":                                        HealthStatusStopping,
		"storage-config-upgrade":                          HealthStatusUpdating,
		"storage-full":                                    HealthStatusUnhealthy,
		"storage-optimization":                            HealthStatusMaintenance,
		"upgrading":                                       HealthStatusUpdating,
	},

	AWSResourceTypeELB: {
		"active":          HealthStatusHealthy,
		"provisioning":    HealthStatusCreating,
		"active_impaired": HealthStatusWarning,
		"failed":          HealthStatusError,
	},

	AWSResourceTypeVPC: {
		"pending":   HealthStatusPending,
		"available": HealthStatusHealthy,
	},

	AWSResourceTypeSubnet: {
		"pending":   HealthStatusPending,
		"available": HealthStatusHealthy,
	},
}
