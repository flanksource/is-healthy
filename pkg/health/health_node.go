package health

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ConditionExpectation struct {
	ExpectedStatus v1.ConditionStatus
	Severity       Health
}

var nodeConditionExpectations = map[string]ConditionExpectation{
	string(v1.NodeNetworkUnavailable): {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthUnhealthy,
	},
	string(v1.NodeDiskPressure): {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	string(v1.NodeMemoryPressure): {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	string(v1.NodePIDPressure): {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"EtcdIsVoter": {
		ExpectedStatus: v1.ConditionTrue,
		Severity:       HealthWarning,
	},
	"CorruptDockerOverlay2": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"KernelDeadlock": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"ReadonlyFilesystem": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"FrequentContainerdRestart": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"FrequentDockerRestart": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"FrequentKubeletRestart": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"FrequentGcfsSnapshotterRestart": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"FrequentGcfsdRestart": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"FrequentUnregisterNetDevice": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"GcfsSnapshotterUnhealthy": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"GcfsdUnhealthy": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"GcfsSnapshotterMissingLayer": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"SecondaryBootDiskMissingLayer": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthWarning,
	},
	"DeprecatedAuthsFieldInContainerdConfiguration": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthHealthy,
	},
	"DeprecatedConfigsFieldInContainerdConfiguration": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthHealthy,
	},
	"DeprecatedMirrorsFieldInContainerdConfiguration": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthHealthy,
	},
	"DeprecatedOtherContainerdFeatures": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthHealthy,
	},
	"DeprecatedPullingSchemaV1Image": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthHealthy,
	},
	"DeprecatedUsingV1Alpha2Cri": {
		ExpectedStatus: v1.ConditionFalse,
		Severity:       HealthHealthy,
	},
}

func getNodeHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var node v1.Node
	if err := convertFromUnstructured(obj, &node); err != nil {
		return nil, err
	}

	hs := HealthStatus{
		Status: HealthStatusCode(node.Status.Phase),
		Health: HealthUnknown,
	}

	switch node.Status.Phase {
	case v1.NodeRunning, "":
		for _, cond := range node.Status.Conditions {
			if cond.Type == v1.NodeReady {
				if cond.Status == v1.ConditionTrue {
					hs.Ready = true
					hs.Health = hs.Health.Worst(HealthHealthy)
					if hs.Health == HealthHealthy {
						hs.Status = HealthStatusRunning
					}
				} else {
					hs.Health = HealthUnhealthy
					hs.Status = HealthStatusCode(HumanCase(string(cond.Type)))
					hs.Message = cond.Message
					return &hs, nil
				}
				continue
			}

			if expectation, exists := nodeConditionExpectations[string(cond.Type)]; exists {
				if cond.Status != expectation.ExpectedStatus {
					newHealth := hs.Health.Worst(expectation.Severity)
					if newHealth.IsWorseThan(hs.Health) {
						hs.Status = HealthStatusCode(HumanCase(string(cond.Type)))
						hs.Message = cond.Message
					}
					hs.Health = newHealth
				}
			}
		}

		for _, taint := range node.Spec.Taints {
			if taint.Key == "node.kubernetes.io/unschedulable" && taint.Effect == "NoSchedule" {
				newHealth := hs.Health.Worst(HealthWarning)
				if newHealth.IsWorseThan(hs.Health) {
					hs.Status = "Unschedulable"
				}
				hs.Health = newHealth
			}
		}
	}

	return &hs, nil
}
