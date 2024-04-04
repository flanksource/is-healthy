/*
Package provides functionality that allows assessing the health state of a Kubernetes resource.
*/

package health_test

import (
	"os"
	"testing"

	"github.com/flanksource/is-healthy/pkg/health"
	"github.com/flanksource/is-healthy/pkg/lua"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func assertAppHealth(t *testing.T, yamlPath string, expectedStatus health.HealthStatusCode) {
	health := getHealthStatus(yamlPath, t)
	assert.NotNil(t, health)
	assert.Equal(t, expectedStatus, health.Status)
}

func getHealthStatus(yamlPath string, t *testing.T) *health.HealthStatus {
	yamlBytes, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	var obj unstructured.Unstructured
	err = yaml.Unmarshal(yamlBytes, &obj)
	require.NoError(t, err)
	health, err := health.GetResourceHealth(&obj, lua.ResourceHealthOverrides{})
	require.NoError(t, err)
	return health
}

func TestNamespace(t *testing.T) {
	assertAppHealth(t, "./testdata/namespace.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/namespace-terminating.yaml", health.HealthStatusDeleting)
}

func TestCertificate(t *testing.T) {
	assertAppHealth(t, "./testdata/certificate-healthy.yaml", health.HealthStatusHealthy)
}

func TestDeploymentHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/nginx.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/deployment-progressing.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/deployment-suspended.yaml", health.HealthStatusSuspended)
	assertAppHealth(t, "./testdata/deployment-degraded.yaml", health.HealthStatusDegraded)
}

func TestStatefulSetHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/statefulset.yaml", health.HealthStatusHealthy)
}

func TestStatefulSetOnDeleteHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/statefulset-ondelete.yaml", health.HealthStatusHealthy)
}

func TestDaemonSetOnDeleteHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/daemonset-ondelete.yaml", health.HealthStatusHealthy)
}
func TestPVCHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/pvc-bound.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/pvc-pending.yaml", health.HealthStatusProgressing)
}

func TestServiceHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/svc-clusterip.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/svc-loadbalancer.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/svc-loadbalancer-unassigned.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/svc-loadbalancer-nonemptylist.yaml", health.HealthStatusHealthy)
}

func TestIngressHealth(t *testing.T) {
	assertAppHealth(t, "./testdata/ingress.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/ingress-unassigned.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/ingress-nonemptylist.yaml", health.HealthStatusHealthy)
}

func TestCRD(t *testing.T) {
	assert.Nil(t, getHealthStatus("./testdata/knative-service.yaml", t))
}

func TestJob(t *testing.T) {
	assertAppHealth(t, "./testdata/job-running.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/job-failed.yaml", health.HealthStatusDegraded)
	assertAppHealth(t, "./testdata/job-succeeded.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/job-suspended.yaml", health.HealthStatusSuspended)
}

func TestHPA(t *testing.T) {
	assertAppHealth(t, "./testdata/hpa-v2-healthy.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/hpa-v2-degraded.yaml", health.HealthStatusDegraded)
	assertAppHealth(t, "./testdata/hpa-v2-progressing.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/hpa-v2beta2-healthy.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/hpa-v2beta1-healthy-disabled.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/hpa-v2beta1-healthy.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/hpa-v1-degraded.yaml", health.HealthStatusDegraded)
	assertAppHealth(t, "./testdata/hpa-v1-healthy.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/hpa-v1-healthy-toofew.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/hpa-v1-progressing.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/hpa-v1-progressing-with-no-annotations.yaml", health.HealthStatusProgressing)
}

func TestPod(t *testing.T) {
	assertAppHealth(t, "./testdata/pod-pending.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/pod-running-not-ready.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/pod-crashloop.yaml", health.HealthStatusDegraded)
	assertAppHealth(t, "./testdata/pod-imagepullbackoff.yaml", health.HealthStatusDegraded)
	assertAppHealth(t, "./testdata/pod-error.yaml", health.HealthStatusDegraded)
	assertAppHealth(t, "./testdata/pod-running-restart-always.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/pod-running-restart-never.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/pod-running-restart-onfailure.yaml", health.HealthStatusProgressing)
	assertAppHealth(t, "./testdata/pod-failed.yaml", health.HealthStatusDegraded)
	assertAppHealth(t, "./testdata/pod-succeeded.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/pod-deletion.yaml", health.HealthStatusProgressing)
}

func TestApplication(t *testing.T) {
	assert.Nil(t, getHealthStatus("./testdata/application-healthy.yaml", t))
	assert.Nil(t, getHealthStatus("./testdata/application-degraded.yaml", t))
}

// func TestAPIService(t *testing.T) {
// 	assertAppHealth(t, "./testdata/apiservice-v1-true.yaml", HealthStatusHealthy)
// 	assertAppHealth(t, "./testdata/apiservice-v1-false.yaml", HealthStatusProgressing)
// 	assertAppHealth(t, "./testdata/apiservice-v1beta1-true.yaml", HealthStatusHealthy)
// 	assertAppHealth(t, "./testdata/apiservice-v1beta1-false.yaml", HealthStatusProgressing)
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
	assertAppHealth(t, "./testdata/argo-application-healthy.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/argo-application-missing.yaml", health.HealthStatusMissing)
}

func TestFluxResources(t *testing.T) {
	assertAppHealth(t, "./testdata/flux-kustomization-healthy.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/flux-kustomization-unhealthy.yaml", health.HealthStatusDegraded)

	assertAppHealth(t, "./testdata/flux-helmrelease-healthy.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/flux-helmrelease-unhealthy.yaml", health.HealthStatusDegraded)

	assertAppHealth(t, "./testdata/flux-helmrepository-healthy.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/flux-helmrepository-unhealthy.yaml", health.HealthStatusDegraded)

	assertAppHealth(t, "./testdata/flux-gitrepository-healthy.yaml", health.HealthStatusHealthy)
	assertAppHealth(t, "./testdata/flux-gitrepository-unhealthy.yaml", health.HealthStatusDegraded)
}
