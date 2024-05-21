package health

import (
	"strings"
)

const (
	AWSResourceTypeEBS    string = "ebs"
	AWSResourceTypeEC2    string = "ec2"
	AWSResourceTypeEKS    string = "eks"
	AWSResourceTypeELB    string = "elb"
	AWSResourceTypeRDS    string = "rds"
	AWSResourceTypeVPC    string = "vpc"
	AWSResourceTypeSubnet string = "subnet"
)

func GetAWSResourceHealth(resourceType, status string) (health HealthStatus) {
	if resourceStatuses, found := awsResourceHealthmap[resourceType]; found {
		if v, found := resourceStatuses[strings.ToLower(status)]; found {
			return v
		}
	}

	return HealthStatus{
		Status: HealthStatusUnknown,
		Health: HealthUnknown,
		Ready:  false,
	}
}

var awsResourceHealthmap = map[string]map[string]HealthStatus{
	AWSResourceTypeEC2: {
		"pending":       HealthStatus{Status: HealthStatusPending, Health: HealthUnknown},
		"running":       HealthStatus{Status: HealthStatusHealthy, Health: HealthHealthy, Ready: true},
		"shutting-down": HealthStatus{Status: HealthStatusDeleting, Health: HealthUnknown},
		"stopped":       HealthStatus{Status: HealthStatusStopped, Health: HealthUnknown},
		"stopping":      HealthStatus{Status: HealthStatusStopping, Health: HealthUnknown},
		"terminated":    HealthStatus{Status: HealthStatusDeleted, Health: HealthUnknown},
	},

	AWSResourceTypeEKS: {
		"creating": HealthStatus{Status: HealthStatusCreating, Health: HealthUnknown},
		"active":   HealthStatus{Status: HealthStatusHealthy, Health: HealthHealthy, Ready: true},
		"deleting": HealthStatus{Status: HealthStatusDeleting, Health: HealthUnknown},
		"failed":   HealthStatus{Status: HealthStatusError, Health: HealthUnhealthy},
		"updating": HealthStatus{Status: HealthStatusUpdating, Health: HealthUnknown},
		"pending":  HealthStatus{Status: HealthStatusPending, Health: HealthUnknown},
	},

	AWSResourceTypeEBS: {
		"creating":  HealthStatus{Status: HealthStatusCreating, Health: HealthUnknown},
		"available": HealthStatus{Status: HealthStatusStopped, Health: HealthHealthy, Ready: true},
		"in-use":    HealthStatus{Status: HealthStatusHealthy, Health: HealthHealthy, Ready: true},
		"deleting":  HealthStatus{Status: HealthStatusDeleting, Health: HealthUnknown},
		"deleted":   HealthStatus{Status: HealthStatusDeleted, Health: HealthUnknown},
		"error":     HealthStatus{Status: HealthStatusError, Health: HealthUnhealthy},
	},

	// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/accessing-monitoring.html
	AWSResourceTypeRDS: {
		"available":                           HealthStatus{Status: HealthStatusHealthy, Health: HealthHealthy, Ready: true},
		"billed":                              HealthStatus{Status: HealthStatusHealthy, Health: HealthHealthy, Ready: true},
		"backing-up":                          HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy},
		"configuring-enhanced-monitoring":     HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy},
		"configuring-iam-database-auth":       HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy},
		"configuring-log-exports":             HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy},
		"converting-to-vpc":                   HealthStatus{Status: HealthStatusUpdating, Health: HealthHealthy, Ready: true},
		"creating":                            HealthStatus{Status: HealthStatusCreating, Health: HealthUnknown},
		"delete-precheck":                     HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy, Ready: true},
		"deleting":                            HealthStatus{Status: HealthStatusDeleting, Health: HealthUnknown},
		"failed":                              HealthStatus{Status: HealthStatusUnhealthy, Health: HealthUnhealthy},
		"inaccessible-encryption-credentials": HealthStatus{Status: HealthStatusInaccesible, Health: HealthUnhealthy},
		"inaccessible-encryption-credentials-recoverable": HealthStatus{Status: HealthStatusInaccesible, Health: HealthWarning},
		"incompatible-network":                            HealthStatus{Status: HealthStatusUnhealthy, Health: HealthUnhealthy},
		"incompatible-option-group":                       HealthStatus{Status: HealthStatusUnhealthy, Health: HealthUnhealthy},
		"incompatible-parameters":                         HealthStatus{Status: HealthStatusUnhealthy, Health: HealthUnhealthy},
		"incompatible-restore":                            HealthStatus{Status: HealthStatusUnhealthy, Health: HealthUnhealthy},
		"insufficient-capacity":                           HealthStatus{Status: HealthStatusUnhealthy, Health: HealthUnhealthy},
		"maintenance":                                     HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy},
		"modifying":                                       HealthStatus{Status: HealthStatusUpdating, Health: HealthHealthy, Ready: true},
		"moving-to-vpc":                                   HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy},
		"rebooting":                                       HealthStatus{Status: HealthStatusRestart, Health: HealthHealthy},
		"resetting-master-credentials":                    HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy, Ready: true},
		"renaming":                                        HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy, Ready: true},
		"restore-error":                                   HealthStatus{Status: HealthStatusError, Health: HealthUnhealthy},
		"starting":                                        HealthStatus{Status: HealthStatusStarting, Health: HealthUnknown},
		"stopped":                                         HealthStatus{Status: HealthStatusStopped, Health: HealthHealthy},
		"stopping":                                        HealthStatus{Status: HealthStatusStopping, Health: HealthUnknown},
		"storage-config-upgrade":                          HealthStatus{Status: HealthStatusUpdating, Health: HealthHealthy, Ready: true},
		"storage-full":                                    HealthStatus{Status: HealthStatusUnhealthy, Health: HealthUnhealthy},
		"storage-optimization":                            HealthStatus{Status: HealthStatusMaintenance, Health: HealthHealthy, Ready: true},
		"upgrading":                                       HealthStatus{Status: HealthStatusUpdating, Health: HealthHealthy},
	},

	AWSResourceTypeELB: {
		"active":          HealthStatus{Status: HealthStatusHealthy, Health: HealthHealthy, Ready: true},
		"provisioning":    HealthStatus{Status: HealthStatusCreating, Health: HealthUnknown},
		"active_impaired": HealthStatus{Status: HealthStatusWarning, Health: HealthWarning, Ready: true},
		"failed":          HealthStatus{Status: HealthStatusError, Health: HealthUnhealthy},
	},

	AWSResourceTypeVPC: {
		"pending":   HealthStatus{Status: HealthStatusPending, Health: HealthUnknown},
		"available": HealthStatus{Status: HealthStatusHealthy, Health: HealthHealthy, Ready: true},
	},

	AWSResourceTypeSubnet: {
		"pending":   HealthStatus{Status: HealthStatusPending, Health: HealthUnknown},
		"available": HealthStatus{Status: HealthStatusHealthy, Health: HealthHealthy, Ready: true},
	},
}
