package health

import (
	"strings"
)

const (
	AWSResourceTypeEBS                 string = "ebs"
	AWSResourceTypeEC2                 string = "ec2"
	AWSResourceTypeEKS                 string = "eks"
	AWSResourceTypeELB                 string = "elb"
	AWSResourceTypeRDS                 string = "rds"
	AWSResourceTypeVPC                 string = "vpc"
	AWSResourceTypeSubnet              string = "subnet"
	AWSResourceTypeCloudformationStack string = "cloudformationstack"
)

func GetAWSResourceHealth(resourceType, status string) (health HealthStatus) {
	if resourceStatuses, found := awsResourceHealthmap[resourceType]; found {
		if v, found := resourceStatuses[strings.ToLower(status)]; found {
			v.Status = HealthStatusCode(lo.Capitalize(strings.ReplaceAll(strings.ReplaceAll(status, "-", " "), "_", " ")))
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
	AWSResourceTypeCloudformationStack: {
		"create_complete":                              HealthStatus{Health: HealthHealthy, Ready: true},
		"create_failed":                                HealthStatus{Health: HealthUnhealthy, Ready: true},
		"create_in_progress":                           HealthStatus{Health: HealthUnknown},
		"delete_complete":                              HealthStatus{Health: HealthUnknown, Ready: true},
		"delete_failed":                                HealthStatus{Health: HealthUnhealthy, Ready: true},
		"delete_in_progress":                           HealthStatus{Health: HealthUnknown},
		"import_complete":                              HealthStatus{Health: HealthHealthy, Ready: true},
		"import_in_progress":                           HealthStatus{Health: HealthUnknown},
		"import_rollback_complete":                     HealthStatus{Health: HealthWarning, Ready: true},
		"import_rollback_failed":                       HealthStatus{Health: HealthUnhealthy, Ready: true},
		"import_rollback_in_progress":                  HealthStatus{Health: HealthWarning},
		"review_in_progress":                           HealthStatus{Health: HealthUnknown},
		"rollback_complete":                            HealthStatus{Health: HealthWarning, Ready: true},
		"rollback_failed":                              HealthStatus{Health: HealthUnhealthy, Ready: true},
		"rollback_in_progress":                         HealthStatus{Health: HealthWarning},
		"update_complete_cleanup_in_progress":          HealthStatus{Health: HealthUnknown},
		"update_complete":                              HealthStatus{Health: HealthHealthy, Ready: true},
		"update_failed":                                HealthStatus{Health: HealthUnhealthy, Ready: true},
		"update_in_progress":                           HealthStatus{Health: HealthUnknown},
		"update_rollback_complete_cleanup_in_progress": HealthStatus{Health: HealthUnknown},
		"update_rollback_complete":                     HealthStatus{Health: HealthWarning, Ready: true},
		"update_rollback_failed":                       HealthStatus{Health: HealthUnhealthy, Ready: true},
		"update_rollback_in_progress":                  HealthStatus{Health: HealthWarning},
	},

	AWSResourceTypeEC2: {
		"pending":       HealthStatus{Health: HealthUnknown},
		"running":       HealthStatus{Health: HealthHealthy, Ready: true},
		"shutting-down": HealthStatus{Health: HealthUnknown},
		"stopped":       HealthStatus{Health: HealthUnknown},
		"stopping":      HealthStatus{Health: HealthUnknown},
		"terminated":    HealthStatus{Health: HealthUnknown},
	},

	AWSResourceTypeEKS: {
		"creating": HealthStatus{Health: HealthUnknown},
		"active":   HealthStatus{Health: HealthHealthy, Ready: true},
		"deleting": HealthStatus{Health: HealthUnknown},
		"failed":   HealthStatus{Health: HealthUnhealthy},
		"updating": HealthStatus{Health: HealthUnknown},
		"pending":  HealthStatus{Health: HealthUnknown},
	},

	AWSResourceTypeEBS: {
		"creating":  HealthStatus{Health: HealthUnknown},
		"available": HealthStatus{Health: HealthHealthy, Ready: true},
		"in-use":    HealthStatus{Health: HealthHealthy, Ready: true},
		"deleting":  HealthStatus{Health: HealthUnknown},
		"deleted":   HealthStatus{Health: HealthUnknown},
		"error":     HealthStatus{Health: HealthUnhealthy},
	},

	// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/accessing-monitoring.html
	AWSResourceTypeRDS: {
		"available":                           HealthStatus{Health: HealthHealthy, Ready: true},
		"billed":                              HealthStatus{Health: HealthHealthy, Ready: true},
		"backing-up":                          HealthStatus{Health: HealthHealthy},
		"configuring-enhanced-monitoring":     HealthStatus{Health: HealthHealthy},
		"configuring-iam-database-auth":       HealthStatus{Health: HealthHealthy},
		"configuring-log-exports":             HealthStatus{Health: HealthHealthy},
		"converting-to-vpc":                   HealthStatus{Health: HealthHealthy, Ready: true},
		"creating":                            HealthStatus{Health: HealthUnknown},
		"delete-precheck":                     HealthStatus{Health: HealthHealthy, Ready: true},
		"deleting":                            HealthStatus{Health: HealthUnknown},
		"failed":                              HealthStatus{Health: HealthUnhealthy},
		"inaccessible-encryption-credentials": HealthStatus{Health: HealthUnhealthy},
		"inaccessible-encryption-credentials-recoverable": HealthStatus{Health: HealthWarning},
		"incompatible-network":                            HealthStatus{Health: HealthUnhealthy},
		"incompatible-option-group":                       HealthStatus{Health: HealthUnhealthy},
		"incompatible-parameters":                         HealthStatus{Health: HealthUnhealthy},
		"incompatible-restore":                            HealthStatus{Health: HealthUnhealthy},
		"insufficient-capacity":                           HealthStatus{Health: HealthUnhealthy},
		"maintenance":                                     HealthStatus{Health: HealthHealthy},
		"modifying":                                       HealthStatus{Health: HealthHealthy, Ready: true},
		"moving-to-vpc":                                   HealthStatus{Health: HealthHealthy},
		"rebooting":                                       HealthStatus{Health: HealthHealthy},
		"resetting-master-credentials":                    HealthStatus{Health: HealthHealthy, Ready: true},
		"renaming":                                        HealthStatus{Health: HealthHealthy, Ready: true},
		"restore-error":                                   HealthStatus{Health: HealthUnhealthy},
		"starting":                                        HealthStatus{Health: HealthUnknown},
		"stopped":                                         HealthStatus{Health: HealthHealthy},
		"stopping":                                        HealthStatus{Health: HealthUnknown},
		"storage-config-upgrade":                          HealthStatus{Health: HealthHealthy, Ready: true},
		"storage-full":                                    HealthStatus{Health: HealthUnhealthy},
		"storage-optimization":                            HealthStatus{Health: HealthHealthy, Ready: true},
		"upgrading":                                       HealthStatus{Health: HealthHealthy},
	},

	AWSResourceTypeELB: {
		"active":          HealthStatus{Health: HealthHealthy, Ready: true},
		"provisioning":    HealthStatus{Health: HealthUnknown},
		"active_impaired": HealthStatus{Health: HealthWarning, Ready: true},
		"failed":          HealthStatus{Health: HealthUnhealthy},
	},

	AWSResourceTypeVPC: {
		"pending":   HealthStatus{Health: HealthUnknown},
		"available": HealthStatus{Health: HealthHealthy, Ready: true},
	},

	AWSResourceTypeSubnet: {
		"pending":   HealthStatus{Health: HealthUnknown},
		"available": HealthStatus{Health: HealthHealthy, Ready: true},
	},
}
