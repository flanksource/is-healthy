package health

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/duration"
)

const defaultMaxMessageLength = 512

var maxMessageLength = defaultMaxMessageLength

var re = regexp.MustCompile(`(?:\((\d+\.?\d*)%\))|(\d+\.?\d*)%`)

const (
	missionControlReadyConditionType = "Ready"

	// See github.com/flanksource/kopper/reconciler.go for the canonical reasons.
	missionControlReasonSynced        = "Synced"
	missionControlReasonPersistFailed = "PersistFailed"
	missionControlReasonDeleteFailed  = "DeleteFailed"
)

func getCanaryHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	errorMsg, _, err := unstructured.NestedString(obj.Object, "status", "errorMessage")
	if err != nil {
		return nil, err
	}

	if errorMsg != "" {
		return &HealthStatus{
			Ready:   true,
			Message: lo.Ellipsis(errorMsg, maxMessageLength),
			Health:  HealthUnhealthy,
		}, nil
	}

	message, _, _ := unstructured.NestedString(obj.Object, "status", "message")
	canaryStatus, _, _ := unstructured.NestedString(obj.Object, "status", "status")
	uptime1h, _, _ := unstructured.NestedString(obj.Object, "status", "uptime1h")

	output := HealthStatus{
		Message: lo.CoalesceOrEmpty(message, fmt.Sprintf("uptime: %s", uptime1h)),
		Status:  HealthStatusCode(canaryStatus),
		Ready:   true,
	}

	switch canaryStatus {
	case "Passed":
		output.Health = HealthHealthy
		if uptime := parseCanaryUptime(uptime1h); uptime != nil && *uptime < float64(80) {
			output.Health = HealthWarning
		}
	case "Failed":
		output.Health = HealthUnhealthy
	case "Invalid":
		output.Health = HealthUnhealthy
		output.Ready = false // needs manual intervention
	}

	return &output, nil
}

func parseCanaryUptime(uptime string) *float64 {
	matches := re.FindStringSubmatch(uptime)
	var matched string
	if len(matches) > 0 {
		if matches[1] != "" {
			matched = matches[1]
		} else if matches[2] != "" {
			matched = matches[2]
		}
	}

	v, err := strconv.ParseFloat(matched, 64)
	if err != nil {
		return nil
	}

	return &v
}

func getScrapeConfigHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	// Use incremental status when present, falling back to lastRun
	statusPath := []string{"status", "lastRun"}
	_, hasIncremental, _ := unstructured.NestedMap(obj.Object, "status", "incremental")
	if hasIncremental {
		statusPath = []string{"status", "incremental"}
	}

	errorCount, _, err := unstructured.NestedInt64(obj.Object, append(statusPath, "error")...)
	if err != nil {
		return nil, err
	}

	successCount, _, err := unstructured.NestedInt64(obj.Object, append(statusPath, "success")...)
	if err != nil {
		return nil, err
	}

	var h Health
	switch {
	case errorCount == 0 && successCount == 0:
		h = HealthUnknown
	case errorCount == 0 && successCount > 0:
		h = HealthHealthy
	case errorCount > 0 && successCount == 0:
		h = HealthUnhealthy
	case errorCount > 0 && successCount > 0:
		h = HealthWarning
	default:
		h = HealthUnknown
	}

	status := &HealthStatus{
		Health: h,
		Ready:  true,
	}

	if errorCount > 0 {
		errorMsgs, _, err := unstructured.NestedStringSlice(obj.Object, append(statusPath, "errors")...)
		if err != nil {
			return nil, err
		}

		if len(errorMsgs) > 0 {
			status.Message = strings.Join(lo.Map(errorMsgs, func(msg string, _ int) string {
				return lo.Ellipsis(msg, maxMessageLength)
			}), ",")
		}
	}

	// Skip staleness check when incremental status is present
	if hasIncremental {
		return status, nil
	}

	if lastRunTime, _, err := unstructured.NestedString(obj.Object, "status", "lastRun", "timestamp"); err != nil {
		return nil, err
	} else if lastRunTime != "" {
		parsedLastRuntime, err := time.Parse(time.RFC3339, lastRunTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse lastRun timestamp: %w", err)
		}

		var nextRuntime time.Time
		if scheduleRaw, _, err := unstructured.NestedString(obj.Object, "spec", "schedule"); err != nil {
			return nil, fmt.Errorf("failed to parse scraper schedule: %w", err)
		} else if scheduleRaw == "" {
			nextRuntime = parsedLastRuntime.Add(time.Hour) // The default schedule
		} else {
			parsedSchedule, err := cron.ParseStandard(scheduleRaw)
			if err != nil {
				return &HealthStatus{
					Health:  HealthUnhealthy,
					Message: fmt.Sprintf("Bad schedule: %s", scheduleRaw),
					Ready:   true,
				}, nil
			}

			nextRuntime = parsedSchedule.Next(parsedLastRuntime)
		}

		// If the ScrapeConfig is few minutes behind the schedule, it's not healthy
		if time.Since(nextRuntime) > time.Minute*10 {
			status.Status = "Stale"
			status.Health = HealthWarning
			status.Message = fmt.Sprintf("scraper hasn't run for %s", duration.HumanDuration(time.Since(parsedLastRuntime)))

			if time.Since(nextRuntime) > time.Hour {
				status.Health = HealthUnhealthy
			}
		}
	}

	return status, nil
}

func getMissionControlHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	if obj.GetKind() == "Notification" {
		return getNotificationHealth(obj)
	}

	status, hasMissionControlSignals, err := getMissionControlConditionHealth(obj)
	if err != nil {
		return nil, err
	}

	if !hasMissionControlSignals {
		return nil, nil
	}

	return status, nil
}

func getMissionControlConditionHealth(obj *unstructured.Unstructured) (*HealthStatus, bool, error) {
	readyCondition := GetGenericStatus(obj).FindCondition(missionControlReadyConditionType)
	_, hasObservedGeneration, err := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	if err != nil {
		return nil, false, err
	}

	hasMissionControlSignals := readyCondition.Type != "" || hasObservedGeneration
	if !hasMissionControlSignals {
		return nil, false, nil
	}

	health := &HealthStatus{
		Health: HealthUnknown,
	}

	switch readyCondition.Status {
	case metav1.ConditionTrue:
		health.Ready = true
		health.Health = HealthHealthy
		health.Status = HealthStatusCode(lo.CoalesceOrEmpty(readyCondition.Reason, missionControlReasonSynced))
	case metav1.ConditionFalse:
		health.Ready = false
		health.Message = readyCondition.Message
		health.Status = HealthStatusCode(readyCondition.Reason)

		switch readyCondition.Reason {
		case missionControlReasonPersistFailed, missionControlReasonDeleteFailed:
			health.Health = HealthUnhealthy
		default:
			health.Health = HealthUnhealthy
		}
	case metav1.ConditionUnknown:
		health.Ready = false
		health.Health = HealthUnknown
		health.Message = readyCondition.Message
		health.Status = HealthStatusCode(lo.CoalesceOrEmpty(readyCondition.Reason, string(HealthStatusUnknown)))
	}

	return health, true, nil
}

func applyFlanksourceObservedGenerationHealth(obj *unstructured.Unstructured, health *HealthStatus) error {
	if health == nil {
		return nil
	}

	group := obj.GroupVersionKind().Group
	if group == "" {
		apiVersion := obj.GetAPIVersion()
		if parts := strings.Split(apiVersion, "/"); len(parts) > 1 {
			group = parts[0]
		}
	}

	if !strings.HasSuffix(group, ".flanksource.com") {
		return nil
	}

	observedGeneration, found, err := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	if err != nil {
		return err
	}

	if !found || observedGeneration >= obj.GetGeneration() {
		return nil
	}

	health.Status = HealthStatusDegraded
	health.Health = HealthWarning
	health.Ready = false

	msg := fmt.Sprintf(
		"observed generation %d is behind metadata generation %d",
		observedGeneration,
		obj.GetGeneration(),
	)
	if !strings.Contains(health.Message, msg) {
		health.PrependMessage("%s", msg)
	}

	return nil
}

func getNotificationHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	conditionHealth, hasMissionControlSignals, err := getMissionControlConditionHealth(obj)
	if err != nil {
		return nil, err
	}

	if hasMissionControlSignals {
		return conditionHealth, nil
	}

	return getNotificationLegacyHealth(obj)
}

func getNotificationLegacyHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	failedCount, _, err := unstructured.NestedInt64(obj.Object, "status", "failed")
	if err != nil {
		return nil, err
	}

	pendingCount, _, err := unstructured.NestedInt64(obj.Object, "status", "pending")
	if err != nil {
		return nil, err
	}

	errorMessage, _, err := unstructured.NestedString(obj.Object, "status", "error")
	if err != nil {
		return nil, err
	}

	sentCount, _, err := unstructured.NestedInt64(obj.Object, "status", "sent")
	if err != nil {
		return nil, err
	}

	status := &HealthStatus{
		Health: HealthUnknown,
		Ready:  true,
	}

	if errorMessage != "" {
		status.Message = errorMessage
		status.Health = HealthUnhealthy
		status.Ready = false
		return status, nil
	}

	if sentCount > 0 {
		status.Health = HealthHealthy
		if failedCount > 0 || pendingCount > 0 {
			status.Health = HealthWarning
		}
	} else {
		if pendingCount > 0 {
			status.Health = HealthWarning
		}
		if failedCount > 0 {
			status.Health = HealthUnhealthy
		}
	}

	// Check lastFailed timestamp
	lastFailedTime, found, err := unstructured.NestedString(obj.Object, "status", "lastFailed")
	if err != nil {
		return nil, err
	}

	if found && lastFailedTime != "" {
		parsedLastFailedTime, err := time.Parse(time.RFC3339, lastFailedTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse lastFailed timestamp: %w", err)
		}

		timeSinceLastFailure := time.Since(parsedLastFailedTime)

		if timeSinceLastFailure <= 12*time.Hour {
			status.Health = HealthWarning
			status.Message = fmt.Sprintf("Failed %s ago", duration.HumanDuration(timeSinceLastFailure))
			if timeSinceLastFailure <= time.Hour {
				status.Health = HealthUnhealthy
			}
		}
	}

	return status, nil
}
