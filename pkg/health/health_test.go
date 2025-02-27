/*
Package provides functionality that allows assessing the health state of a Kubernetes resource.
*/

package health_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/flanksource/is-healthy/pkg/health"
	_ "github.com/flanksource/is-healthy/pkg/lua"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	goyaml "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const RFC3339Micro = "2006-01-02T15:04:05Z"

var (
	_now             = time.Now().UTC()
	defaultOverrides = map[string]string{
		"@now":     _now.Format(RFC3339Micro),
		"@now-1m":  _now.Add(-time.Minute * 1).Format(RFC3339Micro),
		"@now-10m": _now.Add(-time.Minute * 5).Format(RFC3339Micro),
		"@now-15m": _now.Add(-time.Minute * 15).Format(RFC3339Micro),
		"@now-5m":  _now.Add(-time.Minute * 5).Format(RFC3339Micro),
		"@now-1h":  _now.Add(-time.Hour).Format(RFC3339Micro),
		"@now-2h":  _now.Add(-time.Hour * 2).Format(RFC3339Micro),
		"@now-4h":  _now.Add(-time.Hour * 4).Format(RFC3339Micro),
		"@now-8h":  _now.Add(-time.Hour * 8).Format(RFC3339Micro),
		"@now-1d":  _now.Add(-time.Hour * 24).Format(RFC3339Micro),
		"@now-5d":  _now.Add(-time.Hour * 24).Format(RFC3339Micro),
		"@now+10m": _now.Add(time.Minute * 10).Format(RFC3339Micro),
		"@now+5m":  _now.Add(time.Minute * 5).Format(RFC3339Micro),
		"@now+15m": _now.Add(time.Minute * 15).Format(RFC3339Micro),

		"@now+1h": _now.Add(time.Hour).Format(RFC3339Micro),
		"@now+2h": _now.Add(time.Hour * 2).Format(RFC3339Micro),
		"@now+4h": _now.Add(time.Hour * 4).Format(RFC3339Micro),
		"@now+8h": _now.Add(time.Hour * 8).Format(RFC3339Micro),
		"@now+1d": _now.Add(time.Hour * 24).Format(RFC3339Micro),
	}
)

func testFixture(t *testing.T, yamlPath string) {
	t.Run(yamlPath, func(t *testing.T) {
		hr, obj := getHealthStatus(yamlPath, t, make(map[string]string))

		expectedHealth := health.Health(strings.ReplaceAll(filepath.Base(yamlPath), ".yaml", ""))
		if health.IsValidHealth(string(expectedHealth)) {
			assert.Equal(t, expectedHealth, hr.Health)
		}

		if v, ok := obj.GetAnnotations()["expected-health"]; ok {
			assert.Equal(t, health.Health(v), hr.Health)
		}

		if v, ok := obj.GetAnnotations()["expected-message"]; ok {
			assert.Equal(t, v, hr.Message)
		}
		if v, ok := obj.GetAnnotations()["expected-status"]; ok {
			assert.Equal(t, health.HealthStatusCode(v), hr.Status)
		}

		if v, ok := obj.GetAnnotations()["expected-ready"]; ok {
			assert.Equal(t, v == "true", hr.Ready)
		}

		if v, ok := obj.GetAnnotations()["expected-last-update"]; ok {
			if hr.LastUpdated == nil {
				assert.Fail(t, "expected last update but got nil")
			} else {
				assert.Equal(t, v, hr.LastUpdated.Format(time.RFC3339))
			}
		}
	})
}

func TestHealthCompare(t *testing.T) {
	assert.True(t, health.HealthUnhealthy.IsWorseThan(health.HealthWarning))
	assert.Equal(t, health.HealthHealthy, health.HealthHealthy.Worst(health.HealthUnknown))
	assert.Equal(t, health.HealthUnhealthy, health.HealthHealthy.Worst(health.HealthUnhealthy))
}

func assertAppHealthMsg(
	t *testing.T,
	yamlPath string,
	expectedStatus health.HealthStatusCode,
	expectedHealth health.Health,
	expectedReady bool,
	overrides ...string,
) {
	var expectedMsg *string
	if len(overrides) > 0 {
		expectedMsg = lo.ToPtr(overrides[0])
		overrides = overrides[1:]
	}

	m := make(map[string]string)
	for i := 0; i < len(overrides); i += 2 {
		if v, ok := defaultOverrides[overrides[i+1]]; ok {
			m[overrides[i]] = v
		} else {
			m[overrides[i]] = overrides[i+1]
		}
	}
	t.Run(yamlPath, func(t *testing.T) {
		health, _ := getHealthStatus(yamlPath, t, m)
		assert.NotNil(t, health)
		assert.Equal(t, expectedHealth, health.Health)
		assert.Equal(t, expectedReady, health.Ready)
		assert.Equal(t, expectedStatus, health.Status)
		if expectedMsg != nil {
			assert.Equal(t, *expectedMsg, health.Message)
		}
	})
}

func assertAppHealthWithOverwriteMsg(
	t *testing.T,
	yamlPath string,
	overwrites map[string]string,
	expectedStatus health.HealthStatusCode,
	expectedHealth health.Health,
	expectedReady bool,
	expectedMsg string,
) {
	health, _ := getHealthStatus(yamlPath, t, overwrites)

	assert.NotNil(t, health)
	assert.Equal(t, expectedHealth, health.Health)
	assert.Equal(t, expectedReady, health.Ready)
	assert.Equal(t, expectedStatus, health.Status)
	assert.Equal(t, expectedMsg, health.Message)
}

func getHealthStatus(
	yamlPath string,
	t *testing.T,
	overrides map[string]string,
) (*health.HealthStatus, unstructured.Unstructured) {
	if !strings.HasPrefix(yamlPath, "./testdata/") && !strings.HasPrefix(yamlPath, "testdata/") &&
		!strings.HasPrefix(yamlPath, "../resource_customizations") {
		yamlPath = "./testdata/" + yamlPath
	}
	m := make(map[string]string)
	for k, v := range defaultOverrides {
		m[k] = v
	}
	for k, v := range overrides {
		m[k] = v
	}
	var yamlBytes []byte
	var err error

	if strings.Contains(yamlPath, "::") {
		yamlBytes, err = os.ReadFile(strings.ReplaceAll(yamlPath, "::", "/"))
	} else {
		yamlBytes, err = os.ReadFile(yamlPath)
	}
	require.NoError(t, err)

	yamlString := string(yamlBytes)
	keys := lo.Keys(m)
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	for _, k := range keys {
		v := m[k]
		yamlString = strings.ReplaceAll(yamlString, k, v)
	}

	// 2nd iteration, sometimes @now is replaced with @now-5m
	for _, k := range keys {
		v := m[k]
		yamlString = strings.ReplaceAll(yamlString, k, v)
	}

	var obj unstructured.Unstructured
	if !strings.Contains(yamlString, "apiVersion:") && !strings.Contains(yamlString, "kind:") {
		configType := strings.Join(
			strings.Split(strings.ReplaceAll(filepath.Dir(yamlPath), "testdata/", ""), "/"),
			"::",
		)
		var m map[string]any
		err = goyaml.Unmarshal([]byte(yamlString), &m)
		require.NoError(t, err)
		obj = unstructured.Unstructured{Object: m}
		if v, ok := m["annotations"]; ok {
			a := make(map[string]string)
			for k, v := range v.(map[string]any) {
				a[k] = fmt.Sprintf("%s", v)
			}

			obj.SetAnnotations(a)
		}
		return lo.ToPtr(health.GetHealthByConfigType(configType, m)), obj
	}

	err = yaml.Unmarshal([]byte(yamlString), &obj)
	require.NoError(t, err)

	health, err := health.GetResourceHealth(&obj, health.DefaultOverrides)
	require.NoError(t, err)
	return health, obj
}

func TestCrossplane(t *testing.T) {
	assertAppHealthMsg(
		t,
		"./testdata/crossplane-apply-failure.yaml",
		"ApplyFailure",
		health.HealthWarning,
		true,
		"apply failed: an existing `high_availability.0.standby_availability_zone` can only be changed when exchanged with the zone specified in `zone`: ",
	)
	assertAppHealthMsg(t, "./testdata/crossplane-healthy.yaml", "ReconcileSuccess", health.HealthHealthy, true, "")
	assertAppHealthMsg(
		t,
		"./testdata/crossplane-installed.yaml",
		"ActivePackageRevision",
		health.HealthHealthy,
		true,
		"ActivePackageRevision HealthyPackageRevision",
	)
	assertAppHealthMsg(
		t,
		"./testdata/crossplane-provider-revision.yaml",
		"HealthyPackageRevision",
		health.HealthHealthy,
		true,
		"",
	)
	assertAppHealthMsg(
		t,
		"./testdata/crossplane-reconcile-error.yaml",
		"ReconcileError",
		health.HealthUnhealthy,
		true,
		"observe failed: cannot run plan: plan failed: Instance cannot be destroyed: Resource azurerm_kubernetes_cluster_node_pool.prodeu01 has lifecycle.prevent_destroy set, but the plan calls for this resource to be destroyed. To avoid this error and continue with the plan, either disable lifecycle.prevent_destroy or reduce the scope of the plan using the -target flag.",
	)
}

func TestNamespace(t *testing.T) {
	assertAppHealthMsg(t, "./testdata/namespace.yaml", health.HealthStatusHealthy, health.HealthUnknown, true)
	assertAppHealthMsg(
		t,
		"./testdata/namespace-terminating.yaml",
		health.HealthStatusTerminating,
		health.HealthUnknown,
		false,
	)
}

func TestCertificateRequest(t *testing.T) {
	assertAppHealthMsg(t, "./testdata/certificate-request-issued.yaml", "Issued", health.HealthHealthy, true)

	// Approved but then failed
	assertAppHealthMsg(
		t,
		"./testdata/certificate-request-invalid-cluster-issuer.yaml",
		"Pending",
		health.HealthUnhealthy,
		false,
		`Referenced "ClusterIssuer" not found: clusterissuer.cert-manager.io "letsencrypt-staging" not found`,
	)

	// Approved but then failed
	assertAppHealthMsg(t, "./testdata/certificate-request-invalid.yaml", "Failed", health.HealthUnhealthy, true)

	// approved but not issued in 1h
	assertAppHealthMsg(t, "./testdata/certificate-request-pending.yaml", "Pending", health.HealthUnhealthy, false)

	// approved in the last 1h
	assertAppHealthWithOverwriteMsg(t, "./testdata/certificate-request-pending.yaml", map[string]string{
		"2024-10-28T08:22:13Z": time.Now().Add(-time.Minute * 10).Format(time.RFC3339),
	},
		"Pending",
		health.HealthUnknown,
		false,
		`Waiting on certificate issuance from order gitlab/gitlab-registry-tls-1-751983884: "pending"`,
	)
}

func TestCertificate(t *testing.T) {
	// assertAppHealthWithOverwriteMsg(t, "./testdata/certificate-issuing-stuck.yaml", map[string]string{
	// 	"2024-10-28T08:05:00Z": time.Now().Add(-time.Minute * 50).Format(time.RFC3339),
	// }, "IncorrectIssuer", health.HealthWarning, false, `Issuing certificate as Secret was previously issued by "Issuer.cert-manager.io/"`)

	// assertAppHealthWithOverwriteMsg(t, "./testdata/certificate-issuing-stuck.yaml", map[string]string{
	// 	"2024-10-28T08:05:00Z": time.Now().Add(-time.Hour * 2).Format(time.RFC3339),
	// }, "IncorrectIssuer", health.HealthUnhealthy, false, `Issuing certificate as Secret was previously issued by "Issuer.cert-manager.io/"`)

	// assertAppHealthMsg(t, "./testdata/certificate-expired.yaml", "Expired", health.HealthUnhealthy, true)

	// assertAppHealthWithOverwrite(t, "./testdata/about-to-expire.yaml", map[string]string{
	// 	"2024-06-26T12:25:46Z": time.Now().Add(time.Hour).UTC().Format("2006-01-02T15:04:05Z"),
	// }, health.HealthStatusWarning, health.HealthWarning, true)

	assertAppHealthWithOverwriteMsg(t, "./testdata/certificate-renewal.yaml", map[string]string{
		"2025-01-16T14:04:53Z": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),        // not Before
		"2025-01-16T14:09:52Z": time.Now().Add(-time.Minute * 10).UTC().Format(time.RFC3339), // renewal time
	}, "Renewing", health.HealthHealthy, false, "Renewing certificate as renewal was scheduled at 2025-01-16 14:09:47 +0000 UTC")

	assertAppHealthWithOverwriteMsg(t, "./testdata/certificate-renewal.yaml", map[string]string{
		"2025-01-16T14:04:53Z": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), // not Before
		"2025-01-16T14:09:52Z": time.Now().
			Add(-time.Minute * 40).
			UTC().
			Format(time.RFC3339),
		// renewal time over the grace period
	}, "Renewing", health.HealthWarning, false, "Certificate has been in renewal state for > 40m0s")

	assertAppHealthMsg(t, "./testdata/certificate-issuing-first-time.yaml", "Issuing", health.HealthUnknown, false)

	assertAppHealthMsg(
		t,
		"./testdata/certificate-issuing-manually-triggered.yaml",
		"Issuing",
		health.HealthUnknown,
		false,
	)
	assertAppHealthMsg(t, "./testdata/certificate-healthy.yaml", "Issued", health.HealthHealthy, true)

	b := "../resource_customizations/cert-manager.io/Certificate/testdata/"
	assertAppHealthMsg(t, b+"degraded_configError.yaml", "ConfigError", health.HealthUnhealthy, true)
	assertAppHealthMsg(
		t,
		b+"progressing_issuing.yaml",
		"Issuing",
		health.HealthUnknown,
		false,
		"Issuing certificate as Secret does not exist",
	)
}

func TestExternalSecrets(t *testing.T) {
	b := "../resource_customizations/external-secrets.io/ExternalSecret/testdata/"
	assertAppHealthMsg(t, b+"degraded.yaml", "SecretSyncedError", health.HealthUnhealthy, true)
	assertAppHealthMsg(t, b+"progressing.yaml", "Progressing", health.HealthUnknown, false)
	assertAppHealthMsg(t, b+"healthy.yaml", "SecretSynced", health.HealthHealthy, true)
}

func TestStatefulSetHealth(t *testing.T) {
	starting := "./testdata/Kubernetes/StatefulSet/statefulset-starting.yaml"
	assertAppHealthMsg(
		t, starting,
		health.HealthStatusStarting,
		health.HealthUnknown,
		true,
		"0/1 ready",
		"@now",
		"@now-1m",
	)
	assertAppHealthMsg(
		t,
		starting,

		health.HealthStatusStarting,
		health.HealthUnknown,
		true,
		"0/1 ready",
		"@now",
		"@now-5m",
	)
	assertAppHealthMsg(
		t,
		starting,
		health.HealthStatusCrashLoopBackoff,
		health.HealthUnhealthy,
		true,
		"0/1 ready",
		"@now",
		"@now-15m",
	)
	assertAppHealthMsg(
		t,
		starting,
		health.HealthStatusCrashLoopBackoff,
		health.HealthUnhealthy,
		true,
		"0/1 ready",
		"@now",
		"@now-1d",
	)
}

func TestStatefulSetOnDeleteHealth(t *testing.T) {
	assertAppHealthMsg(
		t,
		"./testdata/Kubernetes/StatefulSet/statefulset-ondelete.yaml",
		"TerminatingStalled",
		health.HealthWarning,
		false,
		"terminating for 1d",

		"@now",
		"@now-1d",
	)
}

func TestDaemonSetOnDeleteHealth(t *testing.T) {
	assertAppHealthMsg(t, "./testdata/daemonset-ondelete.yaml", health.HealthStatusRunning, health.HealthHealthy, true)
}

func TestPVCHealth(t *testing.T) {
	assertAppHealthMsg(t, "./testdata/pvc-bound.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealthMsg(t, "./testdata/pvc-pending.yaml", health.HealthStatusProgressing, health.HealthHealthy, false)
}

func TestIngressHealth(t *testing.T) {
	assertAppHealthMsg(t, "./testdata/ingress.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealthMsg(t, "./testdata/ingress-unassigned.yaml", health.HealthStatusPending, health.HealthHealthy, false)
	assertAppHealthMsg(
		t,
		"./testdata/ingress-nonemptylist.yaml",
		health.HealthStatusHealthy,
		health.HealthHealthy,
		true,
	)
}

func TestCRD(t *testing.T) {
	b := "../resource_customizations/serving.knative.dev/Service/testdata/"

	assertAppHealthMsg(t, "./testdata/knative-service.yaml", "Progressing", health.HealthUnknown, false)
	assertAppHealthMsg(t, b+"degraded.yaml", "RevisionFailed", health.HealthUnhealthy, true)
	assertAppHealthMsg(t, b+"healthy.yaml", "", health.HealthHealthy, true)
	assertAppHealthMsg(t, b+"progressing.yaml", "Progressing", health.HealthUnknown, false)
}

func TestCnrmPubSub(t *testing.T) {
	b := "../resource_customizations/pubsub.cnrm.cloud.google.com/PubSubTopic/testdata/"

	assertAppHealthMsg(t, b+"dependency_not_found.yaml", "DependencyNotFound", health.HealthUnhealthy, true)
	assertAppHealthMsg(t, b+"dependency_not_ready.yaml", "DependencyNotReady", health.HealthUnknown, false)
	assertAppHealthMsg(t, b+"up_to_date.yaml", "UpToDate", health.HealthHealthy, true)
	assertAppHealthMsg(t, b+"update_failed.yaml", "UpdateFailed", health.HealthUnhealthy, true)
	assertAppHealthMsg(t, b+"update_in_progress.yaml", "Progressing", health.HealthUnknown, false)
}

func TestHPA(t *testing.T) {
	assertAppHealthMsg(t, "./testdata/hpa-v2-healthy.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealthMsg(t, "./testdata/hpa-v2-degraded.yaml", health.HealthStatusDegraded, health.HealthUnhealthy, false)
	assertAppHealthMsg(
		t,
		"./testdata/hpa-v2-progressing.yaml",
		health.HealthStatusProgressing,
		health.HealthHealthy,
		false,
	)
	assertAppHealthMsg(t, "./testdata/hpa-v2beta2-healthy.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealthMsg(
		t,
		"./testdata/hpa-v2beta1-healthy-disabled.yaml",
		health.HealthStatusHealthy,
		health.HealthHealthy,
		true,
	)
	assertAppHealthMsg(t, "./testdata/hpa-v2beta1-healthy.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealthMsg(t, "./testdata/hpa-v1-degraded.yaml", health.HealthStatusDegraded, health.HealthUnhealthy, false)
	assertAppHealthMsg(t, "./testdata/hpa-v2-degraded.yaml", health.HealthStatusDegraded, health.HealthUnhealthy, false)

	assertAppHealthMsg(t, "./testdata/hpa-v1-healthy.yaml", health.HealthStatusHealthy, health.HealthHealthy, true)
	assertAppHealthMsg(
		t,
		"./testdata/hpa-v1-healthy-toofew.yaml",
		health.HealthStatusHealthy,
		health.HealthHealthy,
		true,
	)
	assertAppHealthMsg(
		t,
		"./testdata/hpa-v1-progressing.yaml",
		health.HealthStatusProgressing,
		health.HealthHealthy,
		false,
	)
	assertAppHealthMsg(
		t,
		"./testdata/hpa-v1-progressing-with-no-annotations.yaml",
		health.HealthStatusProgressing,
		health.HealthHealthy,
		false,
	)
}

// func TestAPIService(t *testing.T) {
// 	assertAppHealthMsg(t, "./testdata/apiservice-v1-true.yaml", HealthStatusHealthy, health.HealthHealthy, true)
// 	assertAppHealthMsg(t, "./testdata/apiservice-v1-false.yaml", HealthStatusProgressing, health.HealthHealthy, true)
// 	assertAppHealthMsg(t, "./testdata/apiservice-v1beta1-true.yaml", HealthStatusHealthy, health.HealthHealthy, true)
// 	assertAppHealthMsg(t, "./testdata/apiservice-v1beta1-false.yaml", HealthStatusProgressing, health.HealthHealthy, true)
// }

func TestGetArgoWorkflowHealth(t *testing.T) {
	sampleWorkflow := unstructured.Unstructured{
		Object: map[string]interface{}{
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

	sampleWorkflow = unstructured.Unstructured{
		Object: map[string]interface{}{
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

	sampleWorkflow = unstructured.Unstructured{
		Object: map[string]interface{}{
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
	assertAppHealthMsg(
		t,
		"./testdata/argo-application-healthy.yaml",
		health.HealthStatusHealthy,
		health.HealthHealthy,
		true,
	)
	assertAppHealthMsg(
		t,
		"./testdata/argo-application-missing.yaml",
		health.HealthStatusMissing,
		health.HealthUnknown,
		false,
	)
}

func TestFluxResources(t *testing.T) {
	assertAppHealthMsg(
		t,
		"./testdata/kustomization-reconciliation-failed.yaml",
		"ReconciliationFailed",
		health.HealthUnhealthy,
		false,
		"CronJob/scale-dev-up namespace not specified: the server could not find the requested resource\n",
	)
	assertAppHealthMsg(
		t,
		"./testdata/kustomization-reconciliation-failed-2.yaml",
		"ReconciliationFailed",
		health.HealthUnhealthy,
		false,
		"HelmRelease/mission-control-agent/atlas-topology dry-run failed: failed to create typed patch object (mission-control-agent/atlas-topology; helm.toolkit.fluxcd.io/v2, Kind=HelmRelease): .spec.chart.spec.targetNamespace: field not declared in schema\n",
	)
	assertAppHealthMsg(
		t,
		"./testdata/flux-kustomization-healthy.yaml",
		"ReconciliationSucceeded",
		health.HealthHealthy,
		true,
	)
	assertAppHealthMsg(t, "./testdata/flux-kustomization-unhealthy.yaml", "Progressing", health.HealthUnknown, false)
	assertAppHealthMsg(t, "./testdata/flux-kustomization-failed.yaml", "BuildFailed", health.HealthUnhealthy, false)
	status, _ := getHealthStatus("./testdata/flux-kustomization-failed.yaml", t, nil)
	assert.Contains(t, status.Message, "err='accumulating resources from 'kubernetes_resource_ingress_fail.yaml'")

	assertAppHealthMsg(
		t,
		"./testdata/flux-helmrelease-healthy.yaml",
		"ReconciliationSucceeded",
		health.HealthHealthy,
		true,
	)
	assertAppHealthMsg(t, "./testdata/flux-helmrelease-unhealthy.yaml", "UpgradeFailed", health.HealthUnhealthy, true)
	assertAppHealthMsg(
		t,
		"./testdata/flux-helmrelease-upgradefailed.yaml",
		"UpgradeFailed",
		health.HealthUnhealthy,
		true,
	)
	helmreleaseStatus, _ := getHealthStatus("./testdata/flux-helmrelease-upgradefailed.yaml", t, nil)
	assert.Contains(
		t,
		helmreleaseStatus.Message,
		"Helm upgrade failed for release mission-control-agent/prod-kubernetes-bundle with chart mission-control-kubernetes@0.1.29: YAML parse error on mission-control-kubernetes/templates/topology.yaml: error converting YAML to JSON: yaml: line 171: did not find expected '-' indicator",
	)
	assert.Equal(t, helmreleaseStatus.Status, health.HealthStatusUpgradeFailed)

	assertAppHealthMsg(t, "./testdata/flux-helmrepository-healthy.yaml", "Succeeded", health.HealthHealthy, true)
	assertAppHealthMsg(t, "./testdata/flux-helmrepository-unhealthy.yaml", "Failed", health.HealthUnhealthy, false)

	assertAppHealthMsg(t, "./testdata/flux-gitrepository-healthy.yaml", "Succeeded", health.HealthHealthy, true)
	assertAppHealthMsg(
		t,
		"./testdata/flux-gitrepository-unhealthy.yaml",
		"GitOperationFailed",
		health.HealthUnhealthy,
		false,
	)
}
