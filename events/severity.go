package events

import "strings"

const (
	SeverityHigh    = "high"
	SeverityInfo    = "info"
	SeverityUnknown = "unknown"
	SeverityLow     = "low"
	SeverityMedium  = "medium"
)

func GetSeverity(event string) string {
	switch event {
	case "AccessPolicy":
		return SeverityLow
	case "AttachVolume":
		return SeverityInfo
	case "BackOff":
		return SeverityHigh
	case "ConntrackFull":
		return SeverityMedium
	case "CreatedExternalResource":
		return SeverityLow
	case "Deleted":
		return SeverityInfo
	case "DeletedExternalResource":
		return SeverityLow
	case "Drain":
		return SeverityLow
	case "EnableMFADevice":
		return SeverityLow
	case "Error":
		return SeverityMedium
	case "Evicted":
		return SeverityMedium
	case "EvictionThresholdMet":
		return SeverityLow
	case "ExceededGracePeriod":
		return SeverityLow
	case "ExternalExpanding":
		return SeverityLow
	case "ExternalProvisioning":
		return SeverityLow
	case "FailedCreatePodSandBox":
		return SeverityLow
	case "FailedDelete":
		return SeverityLow
	case "FailedKillPod":
		return SeverityLow
	case "FailedMount":
		return SeverityLow
	case "FailedPreStopHook":
		return SeverityLow
	case "FailedScheduling":
		return SeverityLow
	case "FailedToUpdateEndpoint":
		return SeverityLow
	case "FailedToUpdateEndpointSlices":
		return SeverityLow
	case "FileSystemResizeRequired":
		return SeverityLow
	case "FileSystemResizeSuccessful":
		return SeverityLow
	case "FreezeScheduled":
		return SeverityLow
	case "InvalidDiskCapacity":
		return SeverityMedium
	case "Killing":
		return SeverityInfo
	case "KubeletIsDown":
		return SeverityHigh
	case "MissingJob":
		return SeverityLow
	case "ModifyLoadBalancerAttributes":
		return SeverityLow
	case "ModifyNetworkInterfaceAttribute":
		return SeverityLow
	case "NetworkNotReady":
		return SeverityHigh
	case "NewArtifact":
		return SeverityLow
	case "NodeAllocatableEnforced":
		return SeverityLow
	case "NodeHasDiskPressure":
		return SeverityHigh
	case "NodeHasInsufficientMemory":
		return SeverityHigh
	case "NodeNotReady":
		return SeverityHigh
	case "NodeNotSchedulable":
		return SeverityHigh
	case "NodeUnderDiskPressure":
		return SeverityLow
	case "NodeUnderMemoryPressure":
		return SeverityMedium
	case "NodeUnreachable":
		return SeverityHigh
	case "NoPods":
		return SeverityLow
	case "NoSourceArtifact":
		return SeverityLow
	case "NotTriggerScaleUp":
		return SeverityLow
	case "NoVMEventScheduled":
		return SeverityLow
	case "OOMKilled":
		return SeverityHigh
	case "OrderCreated":
		return SeverityLow
	case "OrderPending":
		return SeverityLow
	case "PodCrashLooping":
		return SeverityHigh
	case "PreemptScheduled":
		return SeverityLow
	case "Presented":
		return SeverityInfo
	case "ProcessingError":
		return SeverityHigh
	case "Provisioning":
		return SeverityLow
	case "RebootScheduled":
		return SeverityLow
	case "ReconciliationPaused":
		return SeverityLow
	case "RecreatingFailedPod":
		return SeverityLow
	case "RedeployScheduled":
		return SeverityLow
	case "RegisteredNode":
		return SeverityLow
	case "RegisterInstancesWithLoadBalancer":
		return SeverityLow
	case "RELOAD":
		return SeverityLow
	case "RemovingNode":
		return SeverityLow
	case "Requested":
		return SeverityLow
	case "Resizing":
		return SeverityLow
	case "ResourceCreated":
		return SeverityLow
	case "ResourceUpdated":
		return SeverityLow
	case "ScaleDown":
		return SeverityLow
	case "ScaledUpGroup":
		return SeverityLow
	case "ScalingPaused":
		return SeverityLow
	case "ScalingReplicaSet":
		return SeverityLow
	case "ScalingResumed":
		return SeverityLow
	case "SourceUnavailable":
		return SeverityLow
	case "Started":
		return SeverityInfo
	case "Starting":
		return SeverityInfo
	case "Succeeded":
		return SeverityInfo
	case "Success":
		return SeverityInfo
	case "SuccessfulAttachVolume":
		return SeverityInfo
	case "SuccessfulCreate":
		return SeverityInfo
	case "SuccessfulDelete":
		return SeverityInfo
	case "Sync":
		return SeverityInfo
	case "TaintManagerEviction":
		return SeverityLow
	case "TerminateScheduled":
		return SeverityLow
	case "TriggeredScaleUp":
		return SeverityLow
	case "UnexpectedJob":
		return SeverityMedium
	case "Unhealthy":
		return SeverityMedium
	case "Updated":
		return SeverityInfo
	case "UpdatedExternalResource":
		return SeverityLow
	case "UpdatedLoadBalancer":
		return SeverityLow
	case "Upgrade":
		return SeverityLow
	case "VMEventScheduled":
		return SeverityLow
	}

	if strings.HasSuffix(event, "IsDown") {
		return SeverityMedium
	}
	if strings.HasSuffix(event, "NetworkInterface") {
		return SeverityInfo
	}
	if strings.HasSuffix(event, "NotReady") {
		return SeverityLow
	}
	if strings.HasSuffix(event, "Succeeded") {
		return SeverityInfo
	}

	if strings.HasPrefix(event, "Add") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Associate") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Attach") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Authorize") {
		return SeverityMedium
	}
	if strings.HasPrefix(event, "Cannot") {
		return SeverityMedium
	}
	if strings.HasPrefix(event, "Change") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Create") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Delete") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Deleting") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Detach") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Failed") {
		return SeverityMedium
	}
	if strings.HasPrefix(event, "Put") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Revoke") {
		return SeverityMedium
	}
	if strings.HasPrefix(event, "Run") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Tag") {
		return SeverityLow
	}
	if strings.HasPrefix(event, "Update") {
		return SeverityLow
	}

	if strings.Contains(event, "Failed") {
		return SeverityMedium
	}

	return SeverityInfo
}
