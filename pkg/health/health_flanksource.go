package health

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/duration"
)

var re = regexp.MustCompile(`(?:\((\d+\.?\d*)%\))|(\d+\.?\d*)%`)

func getCanaryHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	errorMsg, _, err := unstructured.NestedString(obj.Object, "status", "errorMessage")
	if err != nil {
		return nil, err
	}

	if errorMsg != "" {
		return &HealthStatus{
			Message: errorMsg,
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
	errorCount, _, err := unstructured.NestedInt64(obj.Object, "status", "lastRun", "error")
	if err != nil {
		return nil, err
	}

	successCount, _, err := unstructured.NestedInt64(obj.Object, "status", "lastRun", "success")
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
		errorMsgs, _, err := unstructured.NestedStringSlice(obj.Object, "status", "lastRun", "errors")
		if err != nil {
			return nil, err
		}

		if len(errorMsgs) > 0 {
			status.Message = strings.Join(errorMsgs, ",")
		}
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

func getNotificationHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
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
