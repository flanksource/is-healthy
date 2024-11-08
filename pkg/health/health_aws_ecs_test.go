package health_test

import (
	"testing"

	"github.com/flanksource/is-healthy/pkg/health"
)

func TestECSTask(t *testing.T) {
	assertAppHealthMsg(
		t,
		"AWS::ECS::Task/failed.yaml",
		"CannotPullContainerError",
		health.HealthUnhealthy,
		false,
		"pull image manifest has been retried 5 time(s): failed to resolve ref docker.com/iiab-processing-fargate:dev: failed to do request: Head \"https://docker.com/v2/iiab-processing-fargate/manifests/dev\": dial tcp 10.0.0.1:443: connect: connection refused",
	)
}
