package health_test

import (
	"testing"

	"github.com/flanksource/is-healthy/pkg/health"
)

func TestCnrmContainer(t *testing.T) {
	assertAppHealthMsg(
		t,
		"Kubernetes::ContainerCluster/failed.yaml",
		"UpdateFailed",
		health.HealthUnhealthy,
		true,
		"Update call failed: error applying desired state: summary: googleapi: Error 403: Google Compute Engine: Required 'compute.networks.get' permission for 'projects/flanksource-prod/global/networks/flanksource-workload'.\nDetails:\n[\n  {\n    \"@type\": \"type.googleapis.com/google.rpc.RequestInfo\",\n    \"requestId\": \"0xf1e9e3ca2797eb18\"\n  },\n  {\n    \"@type\": \"type.googleapis.com/google.rpc.ErrorInfo\",\n    \"domain\": \"container.googleapis.com\",\n    \"reason\": \"GCE_PERMISSION_DENIED\"\n  }\n]\n, forbidden",
	)
}
