// Copyright Contributors to the Open Cluster Management project

package transforms

// Tests that verify applyDefaultTransformConfig is correctly applied by the
// refactored specific-kind Builders (ACM-21895).
//
// Each test sets up mergedTransformConfig with a custom field or condition
// flag, calls the Builder with a matching unstructured object, and asserts
// the config-driven property appears in the built node — proving the call to
// applyDefaultTransformConfig is actually wired in.

import (
	"testing"

	ocpapp "github.com/openshift/api/apps/v1"
	gvrpolicy "github.com/stolostron/governance-policy-propagator/api/v1"
	agentv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	appDeployable "github.com/stolostron/multicloud-operators-deployable/pkg/apis/apps/v1"
	acmchannel "open-cluster-management.io/multicloud-operators-channel/pkg/apis/apps/v1"
	acmhelmrelease "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/helmrelease/v1"
	acmrule "github.com/stolostron/multicloud-operators-placementrule/pkg/apis/apps/v1"
	acmapp "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/stretchr/testify/assert"
)

// setupTransformConfig sets mergedTransformConfig with a single custom field
// for the given "Kind.apiGroup" key and returns a teardown func.
func setupTransformConfig(t *testing.T, key string, prop ExtractProperty) func() {
	t.Helper()
	orig := mergedTransformConfig
	mergedTransformConfig = map[string]ResourceConfig{
		key: {properties: []ExtractProperty{prop}},
	}
	return func() { mergedTransformConfig = orig }
}

// setupConditionsConfig enables extractConditions for the given key.
func setupConditionsConfig(t *testing.T, key string) func() {
	t.Helper()
	orig := mergedTransformConfig
	mergedTransformConfig = map[string]ResourceConfig{
		key: {extractConditions: true},
	}
	return func() { mergedTransformConfig = orig }
}

// ---- Deployment -------------------------------------------------------

func TestDeployment_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Deployment.apps", ExtractProperty{
		Name: "strategy", JSONPath: `{.spec.strategy.type}`,
	})()

	d := appsv1.Deployment{TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"spec":       map[string]interface{}{"strategy": map[string]interface{}{"type": "RollingUpdate"}},
	}}

	node := DeploymentResourceBuilder(&d, r).BuildNode()
	assert.Equal(t, "RollingUpdate", node.Properties["strategy"],
		"custom field from CollectorConfig must be extracted by DeploymentResourceBuilder")
}

func TestDeployment_CollectConditions(t *testing.T) {
	defer setupConditionsConfig(t, "Deployment.apps")()

	d := appsv1.Deployment{TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{"type": "Available", "status": "True"},
			},
		},
	}}

	node := DeploymentResourceBuilder(&d, r).BuildNode()
	assert.NotNil(t, node.Properties["condition"],
		"collectConditions must produce a condition property in DeploymentResourceBuilder")
}

// ---- DaemonSet --------------------------------------------------------

func TestDaemonSet_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "DaemonSet.apps", ExtractProperty{
		Name: "updateStrategy", JSONPath: `{.spec.updateStrategy.type}`,
	})()

	d := appsv1.DaemonSet{TypeMeta: metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "DaemonSet",
		"spec": map[string]interface{}{
			"updateStrategy": map[string]interface{}{"type": "OnDelete"},
		},
	}}

	node := DaemonSetResourceBuilder(&d, r).BuildNode()
	assert.Equal(t, "OnDelete", node.Properties["updateStrategy"],
		"custom field from CollectorConfig must be extracted by DaemonSetResourceBuilder")
}

// ---- StatefulSet -------------------------------------------------------

func TestStatefulSet_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "StatefulSet.apps", ExtractProperty{
		Name: "serviceName", JSONPath: `{.spec.serviceName}`,
	})()

	s := appsv1.StatefulSet{TypeMeta: metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"spec":       map[string]interface{}{"serviceName": "my-service"},
	}}

	node := StatefulSetResourceBuilder(&s, r).BuildNode()
	assert.Equal(t, "my-service", node.Properties["serviceName"],
		"custom field from CollectorConfig must be extracted by StatefulSetResourceBuilder")
}

// ---- ReplicaSet -------------------------------------------------------

func TestReplicaSet_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "ReplicaSet.apps", ExtractProperty{
		Name: "minReadySeconds", JSONPath: `{.spec.minReadySeconds}`,
	})()

	rs := appsv1.ReplicaSet{TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "ReplicaSet",
		"spec":       map[string]interface{}{"minReadySeconds": int64(10)},
	}}

	node := ReplicaSetResourceBuilder(&rs, r).BuildNode()
	assert.Equal(t, int64(10), node.Properties["minReadySeconds"],
		"custom field from CollectorConfig must be extracted by ReplicaSetResourceBuilder")
}

// ---- CronJob ----------------------------------------------------------

func TestCronJob_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "CronJob.batch", ExtractProperty{
		Name: "successfulJobsHistoryLimit", JSONPath: `{.spec.successfulJobsHistoryLimit}`,
	})()

	c := batchv1beta1.CronJob{TypeMeta: metav1.TypeMeta{Kind: "CronJob", APIVersion: "batch/v1beta1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "batch/v1beta1",
		"kind":       "CronJob",
		"spec":       map[string]interface{}{"successfulJobsHistoryLimit": int64(5)},
	}}

	node := CronJobResourceBuilder(&c, r).BuildNode()
	assert.Equal(t, int64(5), node.Properties["successfulJobsHistoryLimit"],
		"custom field from CollectorConfig must be extracted by CronJobResourceBuilder")
}

// ---- PersistentVolume -------------------------------------------------

func TestPersistentVolume_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "PersistentVolume", ExtractProperty{
		Name: "storageClassName", JSONPath: `{.spec.storageClassName}`,
	})()

	p := corev1.PersistentVolume{TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume", APIVersion: "v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "PersistentVolume",
		"spec":       map[string]interface{}{"storageClassName": "fast"},
	}}

	node := PersistentVolumeResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "fast", node.Properties["storageClassName"],
		"custom field from CollectorConfig must be extracted by PersistentVolumeResourceBuilder")
}

// ---- Subscription (CRD-backed ACM type) --------------------------------

func TestSubscription_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Subscription.apps.open-cluster-management.io", ExtractProperty{
		Name: "placement", JSONPath: `{.spec.placement.local}`,
	})()

	s := acmapp.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Subscription",
			APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "Subscription",
		"spec": map[string]interface{}{
			"placement": map[string]interface{}{"local": true},
		},
	}}

	node := SubscriptionResourceBuilder(&s, r).BuildNode()
	// JSONPath extracts booleans as strings
	assert.Equal(t, "true", node.Properties["placement"],
		"custom field from CollectorConfig must be extracted by SubscriptionResourceBuilder")
}

// ---- Wildcard kind: collectConditions on all apps resources -----------

func TestDeployment_WildcardGroupConditions(t *testing.T) {
	// Wildcard key "*.apps" enables conditions for all resources in the apps group
	orig := mergedTransformConfig
	mergedTransformConfig = map[string]ResourceConfig{
		"*.apps": {extractConditions: true},
	}
	defer func() { mergedTransformConfig = orig }()

	d := appsv1.Deployment{TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{"type": "Progressing", "status": "True"},
			},
		},
	}}

	node := DeploymentResourceBuilder(&d, r).BuildNode()
	assert.NotNil(t, node.Properties["condition"],
		"wildcard *.apps collectConditions must produce condition property in Deployment")
}

// ---- ConfigPolicyResourceBuilder (unstructured-native) -----------------

func TestConfigPolicy_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"ConfigurationPolicy.policy.open-cluster-management.io",
		ExtractProperty{Name: "remediationAction", JSONPath: `{.spec.remediationAction}`})()

	c := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "policy.open-cluster-management.io/v1",
		"kind":       "ConfigurationPolicy",
		"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
		"spec":       map[string]interface{}{"remediationAction": "enforce"},
	}}

	node := ConfigPolicyResourceBuilder(c).BuildNode()
	assert.Equal(t, "enforce", node.Properties["remediationAction"],
		"custom field from CollectorConfig must be extracted by ConfigPolicyResourceBuilder")
}

// ---- OperatorPolicyResourceBuilder (unstructured-native) ---------------

func TestOperatorPolicy_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"OperatorPolicy.policy.open-cluster-management.io",
		ExtractProperty{Name: "complianceType", JSONPath: `{.spec.complianceType}`})()

	c := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "policy.open-cluster-management.io/v1beta1",
		"kind":       "OperatorPolicy",
		"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
		"spec":       map[string]interface{}{"complianceType": "musthave"},
	}}

	node := OperatorPolicyResourceBuilder(c).BuildNode()
	assert.Equal(t, "musthave", node.Properties["complianceType"],
		"custom field from CollectorConfig must be extracted by OperatorPolicyResourceBuilder")
}

// ---- KyvernoPolicyResourceBuilder (unstructured-native) ----------------

func TestKyvernoPolicy_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"ClusterPolicy.kyverno.io",
		ExtractProperty{Name: "failureAction", JSONPath: `{.spec.validationFailureAction}`})()

	p := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kyverno.io/v1",
		"kind":       "ClusterPolicy",
		"metadata":   map[string]interface{}{"name": "test"},
		"spec":       map[string]interface{}{"validationFailureAction": "enforce"},
	}}

	node := KyvernoPolicyResourceBuilder(p).BuildNode()
	assert.Equal(t, "enforce", node.Properties["failureAction"],
		"custom field from CollectorConfig must be extracted by KyvernoPolicyResourceBuilder")
}

// ---- Verify specific-kind properties still set correctly (regression) --

func TestDeployment_SpecificPropertiesUnaffected(t *testing.T) {
	var desired int32 = 3
	d := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 2,
			Replicas:          3,
			ReadyReplicas:     2,
		},
		Spec: appsv1.DeploymentSpec{Replicas: &desired},
	}

	node := DeploymentResourceBuilder(&d, &unstructured.Unstructured{}).BuildNode()
	assert.Equal(t, int64(2), node.Properties["available"])
	assert.Equal(t, int64(3), node.Properties["current"])
	assert.Equal(t, int64(2), node.Properties["ready"])
	assert.Equal(t, int64(3), node.Properties["desired"])
}

// ---- DeploymentConfig (OpenShift) -------------------------------------

func TestDeploymentConfig_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "DeploymentConfig.apps.openshift.io", ExtractProperty{
		Name: "strategy", JSONPath: `{.spec.strategy.type}`,
	})()

	d := ocpapp.DeploymentConfig{
		TypeMeta: metav1.TypeMeta{Kind: "DeploymentConfig", APIVersion: "apps.openshift.io/v1"},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.openshift.io/v1",
		"kind":       "DeploymentConfig",
		"spec":       map[string]interface{}{"strategy": map[string]interface{}{"type": "Recreate"}},
	}}

	node := DeploymentConfigResourceBuilder(&d, r).BuildNode()
	assert.Equal(t, "Recreate", node.Properties["strategy"],
		"custom field must be extracted by DeploymentConfigResourceBuilder")
}

// ---- ArgoApplication --------------------------------------------------

func TestArgoApplication_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Application.argoproj.io", ExtractProperty{
		Name: "project", JSONPath: `{.spec.project}`,
	})()

	a := ArgoApplication{
		TypeMeta: metav1.TypeMeta{Kind: "Application", APIVersion: "argoproj.io/v1alpha1"},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"spec":       map[string]interface{}{"project": "default"},
	}}

	node := ArgoApplicationResourceBuilder(&a, r).BuildNode()
	assert.Equal(t, "default", node.Properties["project"],
		"custom field must be extracted by ArgoApplicationResourceBuilder")
}

// ---- Channel ----------------------------------------------------------

func TestChannel_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Channel.apps.open-cluster-management.io", ExtractProperty{
		Name: "secretRef", JSONPath: `{.spec.secretRef.name}`,
	})()

	c := acmchannel.Channel{
		TypeMeta: metav1.TypeMeta{
			Kind: "Channel", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "Channel",
		"spec":       map[string]interface{}{"secretRef": map[string]interface{}{"name": "my-secret"}},
	}}

	node := ChannelResourceBuilder(&c, r).BuildNode()
	assert.Equal(t, "my-secret", node.Properties["secretRef"],
		"custom field must be extracted by ChannelResourceBuilder")
}

// ---- AppDeployable ----------------------------------------------------

func TestAppDeployable_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Deployable.apps.open-cluster-management.io", ExtractProperty{
		Name: "deployer", JSONPath: `{.spec.deployer.kind}`,
	})()

	d := appDeployable.Deployable{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployable", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "Deployable",
		"spec":       map[string]interface{}{"deployer": map[string]interface{}{"kind": "Helm"}},
	}}

	node := AppDeployableResourceBuilder(&d, r).BuildNode()
	assert.Equal(t, "Helm", node.Properties["deployer"],
		"custom field must be extracted by AppDeployableResourceBuilder")
}

// ---- AppHelmCR --------------------------------------------------------

func TestAppHelmCR_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "HelmRelease.apps.open-cluster-management.io", ExtractProperty{
		Name: "version", JSONPath: `{.spec.version}`,
	})()

	a := acmhelmrelease.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			Kind: "HelmRelease", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "HelmRelease",
		"spec":       map[string]interface{}{"version": "1.2.3"},
	}}

	node := AppHelmCRResourceBuilder(&a, r).BuildNode()
	assert.Equal(t, "1.2.3", node.Properties["version"],
		"custom field must be extracted by AppHelmCRResourceBuilder")
}

// ---- KlusterletAddonConfig --------------------------------------------

func TestKlusterletAddonConfig_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"KlusterletAddonConfig.agent.open-cluster-management.io",
		ExtractProperty{Name: "clusterName", JSONPath: `{.spec.clusterName}`})()

	p := agentv1.KlusterletAddonConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: "KlusterletAddonConfig", APIVersion: "agent.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "agent.open-cluster-management.io/v1",
		"kind":       "KlusterletAddonConfig",
		"spec":       map[string]interface{}{"clusterName": "my-cluster"},
	}}

	node := KlusterletAddonConfigResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "my-cluster", node.Properties["clusterName"],
		"custom field must be extracted by KlusterletAddonConfigResourceBuilder")
}

// ---- PlacementBinding -------------------------------------------------

func TestPlacementBinding_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"PlacementBinding.apps.open-cluster-management.io",
		ExtractProperty{Name: "bindingOverrides", JSONPath: `{.bindingOverrides.remediationAction}`})()

	p := gvrpolicy.PlacementBinding{
		TypeMeta: metav1.TypeMeta{
			Kind: "PlacementBinding", APIVersion: "apps.open-cluster-management.io/v1",
		},
		PlacementRef: gvrpolicy.PlacementSubject{Name: "my-placement", Kind: "PlacementRule"},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion":       "apps.open-cluster-management.io/v1",
		"kind":             "PlacementBinding",
		"bindingOverrides": map[string]interface{}{"remediationAction": "enforce"},
	}}

	node := PlacementBindingResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "enforce", node.Properties["bindingOverrides"],
		"custom field must be extracted by PlacementBindingResourceBuilder")
}

// ---- PlacementRule ----------------------------------------------------

func TestPlacementRule_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"PlacementRule.apps.open-cluster-management.io",
		ExtractProperty{Name: "clusterConditions", JSONPath: `{.spec.clusterConditions[0].type}`})()

	p := acmrule.PlacementRule{
		TypeMeta: metav1.TypeMeta{
			Kind: "PlacementRule", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "PlacementRule",
		"spec": map[string]interface{}{
			"clusterConditions": []interface{}{
				map[string]interface{}{"type": "ManagedClusterConditionAvailable"},
			},
		},
	}}

	node := PlacementRuleResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "ManagedClusterConditionAvailable", node.Properties["clusterConditions"],
		"custom field must be extracted by PlacementRuleResourceBuilder")
}

// ---- PolicyResourceBuilder (typed) ------------------------------------

func TestPolicy_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"Policy.policy.open-cluster-management.io",
		ExtractProperty{Name: "severity", JSONPath: `{.metadata.annotations.policy\.open-cluster-management\.io/severity}`})()

	p := gvrpolicy.Policy{
		TypeMeta: metav1.TypeMeta{
			Kind: "Policy", APIVersion: "policy.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "policy.open-cluster-management.io/v1",
		"kind":       "Policy",
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"policy.open-cluster-management.io/severity": "high",
			},
		},
	}}

	node := PolicyResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "high", node.Properties["severity"],
		"custom field must be extracted by PolicyResourceBuilder")
}

// ---- CertPolicyResourceBuilder (tests the early-return path) ----------

func TestCertPolicy_CustomFieldFromCollectorConfig_EarlyReturn(t *testing.T) {
	defer setupTransformConfig(t,
		"CertificatePolicy.policy.open-cluster-management.io",
		ExtractProperty{Name: "minimumDuration", JSONPath: `{.spec.minimumDuration}`})()

	// Empty status causes the early return path in CertPolicyResourceBuilder
	c := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "policy.open-cluster-management.io/v1",
		"kind":       "CertificatePolicy",
		"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
		"spec":       map[string]interface{}{"minimumDuration": "300h"},
	}}

	node := CertPolicyResourceBuilder(c).BuildNode()
	assert.Equal(t, "300h", node.Properties["minimumDuration"],
		"custom field must be extracted even via the early-return path of CertPolicyResourceBuilder")
}

// ---- PolicyReportResourceBuilder --------------------------------------

func TestPolicyReport_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "PolicyReport.wgpolicyk8s.io", ExtractProperty{
		Name: "reportType", JSONPath: `{.metadata.labels.app\.kubernetes\.io/name}`,
	})()

	pr := PolicyReport{
		TypeMeta: metav1.TypeMeta{Kind: "PolicyReport", APIVersion: "wgpolicyk8s.io/v1alpha2"},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "wgpolicyk8s.io/v1alpha2",
		"kind":       "PolicyReport",
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				"app.kubernetes.io/name": "kyverno",
			},
		},
	}}

	node := PolicyReportResourceBuilder(&pr, r).BuildNode()
	assert.Equal(t, "kyverno", node.Properties["reportType"],
		"custom field must be extracted by PolicyReportResourceBuilder")
}

// ---- additionalColumns pass-through (CRD-backed kind) -----------------

func TestSubscription_AdditionalColumnsPassThrough(t *testing.T) {
	// Verify that CRD additionalPrinterColumns from the informer flow through
	// to the node when a CollectorConfig sets additionalPrinterColumnsPriority.
	priority := 0
	orig := mergedTransformConfig
	mergedTransformConfig = map[string]ResourceConfig{
		"Subscription.apps.open-cluster-management.io": {
			additionalPrinterColumnsPriority: &priority,
		},
	}
	defer func() { mergedTransformConfig = orig }()

	s := acmapp.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind: "Subscription", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "Subscription",
	}}
	additionalCol := ExtractProperty{Name: "phase", JSONPath: `{.status.phase}`}
	r.Object["status"] = map[string]interface{}{"phase": "Subscribed"}

	node := SubscriptionResourceBuilder(&s, r, additionalCol).BuildNode()
	assert.Equal(t, "Subscribed", node.Properties["phase"],
		"additionalColumns must flow through SubscriptionResourceBuilder variadic param")
}
