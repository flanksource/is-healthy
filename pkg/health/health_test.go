/*
Package provides functionality that allows assessing the health state of a Kubernetes resource.
*/

package health_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/flanksource/is-healthy/pkg/health"
	"github.com/flanksource/is-healthy/pkg/lua"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func assertAppHealthMsg(t *testing.T, yamlPath string, expectedStatus health.HealthStatusCode, expectedHealth health.Health, expectedReady bool, expectedMsg string) {
	health := getHealthStatus(yamlPath, t, nil)
	assert.NotNil(t, health)
	assert.Equal(t, expectedHealth, health.Health)
	assert.Equal(t, expectedReady, health.Ready)
	assert.Equal(t, expectedStatus, health.Status)
	assert.Equal(t, expectedMsg, health.Message)
}

func assertAppHealth(t *testing.T, yamlPath string, expectedStatus health.HealthStatusCode, expectedHealth health.Health, expectedReady bool) {
	health := getHealthStatus(yamlPath, t, nil)
	assert.NotNil(t, health)
	assert.Equal(t, expectedHealth, health.Health)
	assert.Equal(t, expectedReady, health.Ready)
	assert.Equal(t, expectedStatus, health.Status)
}

func assertAppHealthWithOverwriteMsg(t *testing.T, yamlPath string, overwrites map[string]string, expectedStatus health.HealthStatusCode, expectedHealth health.Health, expectedReady bool, expectedMsg string) {
	health := getHealthStatus(yamlPath, t, overwrites)
	assert.NotNil(t, health)
	assert.Equal(t, expectedHealth, health.Health)
	assert.Equal(t, expectedReady, health.Ready)
	assert.Equal(t, expectedStatus, health.Status)
	assert.Equal(t, expectedMsg, health.Message)
}

func assertAppHealthWithOverwrite(t *testing.T, yamlPath string, overwrites map[string]string, expectedStatus health.HealthStatusCode, expectedHealth health.Health, expectedReady bool) {
	health := getHealthStatus(yamlPath, t, overwrites)
	assert.NotNil(t, health)
	assert.Equal(t, expectedHealth, health.Health)
	assert.Equal(t, expectedReady, health.Ready)
	assert.Equal(t, expectedStatus, health.Status)
}

func getHealthStatus(yamlPath string, t *testing.T, overwrites map[string]string) *health.HealthStatus {
	yamlBytes, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	// Basic, search & replace overwrite
	for k, v := range overwrites {
		yamlBytes = []byte(strings.ReplaceAll(string(yamlBytes), k, v))
	}

	var obj unstructured.Unstructured
	err = yaml.Unmarshal(yamlBytes, &obj)
	require.NoError(t, err)

	health, err := health.GetResourceHealth(&obj, lua.ResourceHealthOverrides{})
	require.NoError(t, err)
	return health
}

func TestCrossplane(t *testing.T) {
	assertAppHealthMsg(t, "./testdata/crossplane-apply-failure.yaml", "ApplyFailure", health.HealthWarning, true, "apply failed: an existing `high_availability.0.standby_availability_zone` can only be changed when exchanged with the zone specified in `zone`: ")
	assertAppHealthMsg(t, "./testdata/crossplane-healthy.yaml", "ReconcileSuccess", health.HealthHealthy, true, "")
	assertAppHealthMsg(t, "./testdata/crossplane-installed.yaml", "ActivePackageRevision", health.HealthHealthy, true, "")
	assertAppHealthMsg(t, "./testdata/crossplane-provider-revision.yaml", "HealthyPackageRevision", health.HealthHealthy, true, "")
	assertAppHealthMsg(t, "./testdata/crossplane-reconcile-error.yaml", "ReconcileError", health.HealthWarning, true, "observe failed: cannot run plan: plan failed: Instance cannot be destroyed: Resource azurerm_kubernetes_cluster_node_pool.prodeu01 has lifecycle.prevent_destroy set, but the plan calls for this resource to be destroyed. To avoid this error and continue with the plan, either disable lifecycle.prevent_destroy or reduce the scope of the plan using the -target flag.")
}

func TestNamespace(t *testing.T) {
	assertAppHealth(t, "./testdata/namespace.yaml", health.HealthStatusHealthy, health.HealthUnknown, true)
	assertAppHealth(t, "./testdata/namespace-terminating.yaml", health.HealthStatusTerminating, health.HealthUnknown, false)
}

func TestCertificate(t *testing.T) {
	assertAppHealth(t, "./testdata/about-to-expire.yaml", "Warning", health.HealthWarning, true)
	assertAppHealth(t, "./testdata/certificate-healthy.yaml", "Issued", health.HealthHealthy, true)
	b := "../resource_customizations/cert-manager.io/Certificate/testdata/"
	assertAppHealth(t, b+"degraded_configError.yaml", "ConfigError", health.HealthUnhealthy, true)
	assertAppHealth(t, b+"progressing_issuing.yaml", "Issuing", health.HealthUnknown, false)
}

func TestExternalSecrets(t *testing.T) {
	b := "../resource_customizations/external-secrets.io/ExternalSecret/testdata/"
	assertAppHealth(t, b+"degraded.yaml", "", health.HealthUnhealthy, true)
	assertAppHealth(t, b+"progressing.yaml", "Progressing", health.HealthUnknown, false)
	assertAppHealth(t, b+"healthy.yaml", "", health.HealthHealthy, true)
}

func TestDeploymentHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/nginx.yaml", health.HealthStatusRunning, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/deployment-progressing.yaml", health.HealthStatusStarting, health.HealthHealthy, false)
	assertAppHealth(t, "./testdata/deployment-suspended.yaml", health.HealthStatusSuspended, health.HealthHealthy, false)
	assertAppHealth(t, "./testdata/deployment-degraded.yaml", health.HealthStatusStarting, health.HealthHealthy, false)
	assertAppHealth(t, "./testdata/deployment-scaling-down.yaml", health.HealthStatusScalingDown, health.HealthHealthy, false)
	assertAppHealth(t, "./testdata/deployment-failed.yaml", health.HealthStatusRolloutFailed, health.HealthUnhealthy, false)
}

func TestStatefulSetHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/statefulset.yaml", health.HealthStatusRollingOut, health.HealthWarning, false)
}

func TestStatefulSetOnDeleteHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/statefulset-ondelete.yaml", health.HealthStatusRollingOut, health.HealthWarning, false)
}

func TestDaemonSetOnDeleteHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/daemonset-ondelete.yaml", health.HealthStatusRunning, health.HealthHealthy, true)
}
func TestPVCHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/pvc-bound.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/pvc-pending.yaml", health.HealthStatusProgressing, health.HealthHealthy, false)
}

func TestServiceHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/svc-clusterip.yaml", health.HealthStatusUnknown, health.HealthUnknown, true)
	assertAppHealth(t, "./testdata/svc-loadbalancer.yaml", health.HealthStatusRunning, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/svc-loadbalancer-unassigned.yaml", health.HealthStatusCreating, health.HealthUnknown, false)
	assertAppHealth(t, "./testdata/svc-loadbalancer-nonemptylist.yaml", health.HealthStatusRunning, health.HealthHealthy, true)
}

func TestIngressHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/ingress.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/ingress-unassigned.yaml", health.HealthStatusPending, health.HealthHealthy, false)
	assertAppHealth(t, "./testdata/ingress-nonemptylist.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
}

func TestCRD(t *testing.T) {
	assertAppHealth(t, "./testdata/knative-service.yaml", health.HealthStatusProgressing, health.HealthUnknown, false)
}

func TestJob(t *testing.T) {
	assertAppHealth(t, "./testdata/job-running.yaml", health.HealthStatusRunning, health.HealthHealthy, false)
	assertAppHealth(t, "./testdata/job-failed.yaml", health.HealthStatusError, health.HealthUnhealthy, true)
	assertAppHealth(t, "./testdata/job-succeeded.yaml", health.HealthStatusCompleted, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/job-suspended.yaml", health.HealthStatusSuspended, health.HealthUnknown, false)
}

func TestHPA(t *testing.T) {
	assertAppHealth(t, "./testdata/hpa-v2-healthy.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/hpa-v2-degraded.yaml", health.HealthStatusDegraded, health.HealthUnhealthy, false)
	assertAppHealth(t, "./testdata/hpa-v2-progressing.yaml", health.HealthStatusProgressing, health.HealthHealthy, false)
	assertAppHealth(t, "./testdata/hpa-v2beta2-healthy.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/hpa-v2beta1-healthy-disabled.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/hpa-v2beta1-healthy.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/hpa-v1-degraded.yaml", health.HealthStatusDegraded, health.HealthUnhealthy, false)
	assertAppHealth(t, "./testdata/hpa-v2-degraded.yaml", health.HealthStatusDegraded, health.HealthUnhealthy, false)

	assertAppHealth(t, "./testdata/hpa-v1-healthy.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/hpa-v1-healthy-toofew.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/hpa-v1-progressing.yaml", health.HealthStatusProgressing, health.HealthHealthy, false)
	assertAppHealth(t, "./testdata/hpa-v1-progressing-with-no-annotations.yaml", health.HealthStatusProgressing, health.HealthHealthy, false)
}

func TestPod(t *testing.T) {
	assertAppHealth(t, "./testdata/terminating-stuck.yaml", "TerminatingStalled", health.HealthUnhealthy, false)
	assertAppHealth(t, "./testdata/terminating-namespace.yaml", "TerminatingStalled", health.HealthUnhealthy, false)

	assertAppHealthWithOverwrite(t, "./testdata/pod-terminating.yaml", map[string]string{
		"2024-07-01T06:52:22Z": time.Now().Add(-time.Minute * 20).UTC().Format("2006-01-02T15:04:05Z"),
	}, health.HealthStatusTerminating, health.HealthWarning, false)

	assertAppHealthWithOverwrite(t, "./testdata/pod-deletion.yaml", map[string]string{
		"2018-12-03T10:16:04Z": time.Now().Add(-time.Minute).Format("2006-01-02T15:04:05Z"),
	}, health.HealthStatusTerminating, health.HealthUnknown, false)

	assertAppHealthWithOverwriteMsg(t, "./testdata/pod-not-ready-container-not-ready.yaml", map[string]string{
		"2024-07-29T06:32:56Z": time.Now().Add(time.Minute * 10).Format(time.RFC3339),
	}, health.HealthStatusStarting, health.HealthUnknown, false, "Container nginx is waiting for readiness probe")

	// Pod not ready
	assertAppHealth(t, "./testdata/pod-not-ready-but-container-ready.yaml", health.HealthStatusRunning, health.HealthWarning, false)

	// Restart Loop
	assertAppHealth(t, "./testdata/pod-ready-container-terminated.yaml", health.HealthStatusRunning, health.HealthHealthy, true)

	assertAppHealthWithOverwrite(t, "./testdata/pod-ready-container-terminated.yaml", map[string]string{
		"2024-07-18T12:03:06Z": time.Now().Add(-time.Minute * 50).UTC().Format("2006-01-02T15:04:05Z"), // container last terminated
	}, health.HealthStatusRunning, health.HealthWarning, false)

	// Less than 30 minutes
	assertAppHealthWithOverwrite(t, "./testdata/pod-high-restart-count.yaml", map[string]string{
		"2024-07-17T14:29:51Z": time.Now().UTC().Add(-time.Minute).Format("2006-01-02T15:04:05Z"),
		"2024-07-17T14:29:52Z": time.Now().UTC().Add(-time.Minute).Format("2006-01-02T15:04:05Z"), // start time
	}, "OOMKilled", health.HealthUnhealthy, false)

	// Less than 8 hours
	assertAppHealthWithOverwrite(t, "./testdata/pod-high-restart-count.yaml", map[string]string{
		"2024-07-17T14:29:51Z": time.Now().UTC().Add(-time.Hour).Format("2006-01-02T15:04:05Z"),
		"2024-07-17T14:29:52Z": time.Now().UTC().Add(-time.Hour).Format("2006-01-02T15:04:05Z"), // start time
	}, "OOMKilled", health.HealthWarning, false)

	// More than 8 hours
	assertAppHealthWithOverwrite(t, "./testdata/pod-high-restart-count.yaml", map[string]string{
		"2024-07-17T14:29:51Z": "2024-06-17T14:29:51Z",
	}, health.HealthStatusRunning, health.HealthHealthy, true)

	assertAppHealth(t, "./testdata/pod-old-restarts.yaml", health.HealthStatusRunning, health.HealthHealthy, true)

	assertAppHealth(t, "./testdata/pod-pending.yaml", health.HealthStatusPending, health.HealthUnknown, false)
	assertAppHealth(t, "./testdata/pod-running-not-ready.yaml", health.HealthStatusStarting, health.HealthUnknown, false)
	assertAppHealth(t, "./testdata/pod-crashloop.yaml", health.HealthStatusCrashLoopBackoff, health.HealthUnhealthy, false)
	assertAppHealth(t, "./testdata/pod-crashloop-pending.yaml", health.HealthStatusCrashLoopBackoff, health.HealthUnhealthy, false)
	assertAppHealth(t, "./testdata/pod-imagepullbackoff.yaml", "ImagePullBackOff", health.HealthUnhealthy, false)
	assertAppHealth(t, "./testdata/pod-error.yaml", health.HealthStatusError, health.HealthUnhealthy, true)
	assertAppHealth(t, "./testdata/pod-running-restart-always.yaml", health.HealthStatusRunning, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/pod-running-restart-never.yaml", health.HealthStatusRunning, health.HealthHealthy, false)
	assertAppHealth(t, "./testdata/pod-running-restart-onfailure.yaml", health.HealthStatusRunning, health.HealthUnhealthy, false)
	assertAppHealth(t, "./testdata/pod-failed.yaml", health.HealthStatusError, health.HealthUnhealthy, true)
	assertAppHealth(t, "./testdata/pod-succeeded.yaml", health.HealthStatusCompleted, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/pod-init-container-fail.yaml", health.HealthStatusCrashLoopBackoff, health.HealthUnhealthy, false)
}

// func TestAPIService(t *testing.T) {
// 	assertAppHealth(t, "./testdata/apiservice-v1-true.yaml", HealthStatusHealthy, health.HealthHealthy, true)
// 	assertAppHealth(t, "./testdata/apiservice-v1-false.yaml", HealthStatusProgressing, health.HealthHealthy, true)
// 	assertAppHealth(t, "./testdata/apiservice-v1beta1-true.yaml", HealthStatusHealthy, health.HealthHealthy, true)
// 	assertAppHealth(t, "./testdata/apiservice-v1beta1-false.yaml", HealthStatusProgressing, health.HealthHealthy, true)
// }

func TestGetArgoWorkflowHealth(t *testing.T) {
	sampleWorkflow := unstructured.Unstructured{Object: map[string]interface{}{
		"spec": map[string]interface{}{
			"entrypoint":    "sampleEntryPoint",
			"extraneousKey": "we are agnostic to extraneous keys",
		},
		"status": map[string]interface{}{
			"phase":   "Running",
			"message": "This node is running",
		},
	},
	}

	argohealth, err := health.GetArgoWorkflowHealth(&sampleWorkflow)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatusProgressing, argohealth.Status)
	assert.Equal(t, "This node is running", argohealth.Message)

	sampleWorkflow = unstructured.Unstructured{Object: map[string]interface{}{
		"spec": map[string]interface{}{
			"entrypoint":    "sampleEntryPoint",
			"extraneousKey": "we are agnostic to extraneous keys",
		},
		"status": map[string]interface{}{
			"phase":   "Succeeded",
			"message": "This node is has succeeded",
		},
	},
	}

	argohealth, err = health.GetArgoWorkflowHealth(&sampleWorkflow)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatusHealthy, argohealth.Status)
	assert.Equal(t, "This node is has succeeded", argohealth.Message)

	sampleWorkflow = unstructured.Unstructured{Object: map[string]interface{}{
		"spec": map[string]interface{}{
			"entrypoint":    "sampleEntryPoint",
			"extraneousKey": "we are agnostic to extraneous keys",
		},
	},
	}

	argohealth, err = health.GetArgoWorkflowHealth(&sampleWorkflow)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatusProgressing, argohealth.Status)
	assert.Equal(t, "", argohealth.Message)

}

func TestArgoApplication(t *testing.T) {
	assertAppHealth(t, "./testdata/argo-application-healthy.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/argo-application-missing.yaml", health.HealthStatusMissing, health.HealthUnknown, false)
}

func TestFluxResources(t *testing.T) {
	assertAppHealthMsg(t, "./testdata/kustomization-reconciliation-failed.yaml", "ReconciliationFailed", health.HealthUnhealthy, false, "CronJob/scale-dev-up namespace not specified: the server could not find the requested resource\n")
	assertAppHealthMsg(t, "./testdata/kustomization-reconciliation-failed-2.yaml", "ReconciliationFailed", health.HealthUnhealthy, false, "HelmRelease/mission-control-agent/atlas-topology dry-run failed: failed to create typed patch object (mission-control-agent/atlas-topology; helm.toolkit.fluxcd.io/v2, Kind=HelmRelease): .spec.chart.spec.targetNamespace: field not declared in schema\n")
	assertAppHealth(t, "./testdata/flux-kustomization-healthy.yaml", "ReconciliationSucceeded", health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/flux-kustomization-unhealthy.yaml", "Progressing", health.HealthUnknown, false)
	assertAppHealth(t, "./testdata/flux-kustomization-failed.yaml", "BuildFailed", health.HealthUnhealthy, false)
	status := getHealthStatus("./testdata/flux-kustomization-failed.yaml", t, nil)
	assert.Contains(t, status.Message, "err='accumulating resources from 'kubernetes_resource_ingress_fail.yaml'")

	assertAppHealth(t, "./testdata/flux-helmrelease-healthy.yaml", "ReconciliationSucceeded", health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/flux-helmrelease-unhealthy.yaml", "UpgradeFailed", health.HealthUnhealthy, true)
	assertAppHealth(t, "./testdata/flux-helmrelease-upgradefailed.yaml", "UpgradeFailed", health.HealthUnhealthy, true)
	helmreleaseStatus := getHealthStatus("./testdata/flux-helmrelease-upgradefailed.yaml", t, nil)
	assert.Contains(t, helmreleaseStatus.Message, "Helm upgrade failed for release mission-control-agent/prod-kubernetes-bundle with chart mission-control-kubernetes@0.1.29: YAML parse error on mission-control-kubernetes/templates/topology.yaml: error converting YAML to JSON: yaml: line 171: did not find expected '-' indicator")
	assert.Equal(t, helmreleaseStatus.Status, health.HealthStatusUpgradeFailed)

	assertAppHealth(t, "./testdata/flux-helmrepository-healthy.yaml", "Succeeded", health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/flux-helmrepository-unhealthy.yaml", "Failed", health.HealthUnhealthy, false)

	assertAppHealth(t, "./testdata/flux-gitrepository-healthy.yaml", "Succeeded", health.HealthHealthy, true)
	assertAppHealth(t, "./testdata/flux-gitrepository-unhealthy.yaml", "GitOperationFailed", health.HealthUnhealthy, false)
}
