package transforms

import (
	"context"

	"github.com/stolostron/search-collector/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

// Declares a property to extract from a resource using jsonpath.
type ExtractProperty struct {
	Name     string   // `json:"name,omitempty"`
	JSONPath string   // `json:"jsonpath,omitempty"`
	DataType DataType // `json:"dataType,omitempty"`
	// A property to denote that we should only extract this property if this label matches the resource FUTURE: generalize with matchExpression
	matchLabel string // `json:"matchLabel,omitempty"`
	// An internal property to denote this property should be set on the node's metadata instead.
	metadataOnly bool
}

type ExtractEdge struct {
	Name   string   // `json:"name,omitempty"`
	ToKind string   // `json:"toKind,omitempty"`
	Type   EdgeType // `json:"type,omitempty"`
}

type DataType string

const (
	DataTypeBytes     DataType = "bytes"
	DataTypeSlice     DataType = "slice"
	DataTypeString    DataType = "string"
	DataTypeNumber    DataType = "number"
	DataTypeMapString DataType = "mapString"
)

func stringToDataType(s string) DataType {
	switch s {
	case "DataTypeBytes":
		return DataTypeBytes
	case "DataTypeSlice":
		return DataTypeSlice
	case "DataTypeString":
		return DataTypeString
	case "DataTypeNumber":
		return DataTypeNumber
	case "DataTypeMapString":
		return DataTypeMapString
	default:
		return DataTypeString
	}
}

// Declares the properties to extract from a given resource.
type ResourceConfig struct {
	properties         []ExtractProperty // `json:"properties,omitempty"`
	edges              []ExtractEdge     // `json:"edges,omitempty"`
	extractAnnotations bool              // `json:"extractAnnotations,omitempty"`
	extractConditions  bool              // `json:"extractConditions,omitempty"`
}

var (
	defaultTransformIgnoredFields = map[string]bool{
		// Skip age since this likely duplicates "created" that is set by genericProperties.
		"age": true,
	}
	knownStringArrays = map[string]bool{
		"accessMode": true,
		"category":   true,
		"container":  true,
		"image":      true,
		"port":       true,
		"role":       true,
		"rules":      true,
		"subject":    true,
	}
)

// Declares properties to extract from the resource by default. Extended by configurable collection CR searchcollectorcustomizableconfig
var defaultTransformConfig = map[string]ResourceConfig{
	"ClusterServiceVersion.operators.coreos.com": {
		properties: []ExtractProperty{
			{Name: "version", JSONPath: "{.spec.version}"},
			{Name: "display", JSONPath: "{.spec.displayName}"},
			{Name: "phase", JSONPath: "{.status.phase}"},
		},
	},
	"ClusterOperator.config.openshift.io": {
		properties: []ExtractProperty{
			{Name: "version", JSONPath: `{.status.versions[?(@.name=="operator")].version}`},
			{Name: "available", JSONPath: `{.status.conditions[?(@.type=="Available")].status}`},
			{Name: "progressing", JSONPath: `{.status.conditions[?(@.type=="Progressing")].status}`},
			{Name: "degraded", JSONPath: `{.status.conditions[?(@.type=="Degraded")].status}`},
		},
	},
	"ConfigMap": {
		properties: []ExtractProperty{
			{Name: "configParamMaxDesiredLatency", JSONPath: `{.data.spec\.param\.maxDesiredLatencyMilliseconds}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configParamNADNamespace", JSONPath: `{.data.spec\.param\.networkAttachmentDefinitionNamespace}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configParamNADName", JSONPath: `{.data.spec\.param\.networkAttachmentDefinitionName}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configParamTargetNode", JSONPath: `{.data.spec\.param\.targetNode}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configParamSourceNode", JSONPath: `{.data.spec\.param\.sourceNode}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configParamSampleDuration", JSONPath: `{.data.spec\.param\.sampleDurationSeconds}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configTimeout", JSONPath: `{.data.spec\.timeout}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configCompletionTimestamp", JSONPath: `{.data.status\.completionTimestamp}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configFailureReason", JSONPath: `{.data.status\.failureReason}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configStartTimestamp", JSONPath: `{.data.status\.startTimestamp}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configSucceeded", JSONPath: `{.data.status\.succeeded}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configStatusAVGLatencyNano", JSONPath: `{.data.status\.result\.avgLatencyNanoSec}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configStatusMaxLatencyNano", JSONPath: `{.data.status\.result\.maxLatencyNanoSec}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configStatusMinLatencyNano", JSONPath: `{.data.status\.result\.minLatencyNanoSec}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configStatusMeasurementDuration", JSONPath: `{.data.status\.result\.measurementDurationSec}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configStatusTargetNode", JSONPath: `{.data.status\.result\.targetNode}`, matchLabel: "kiagnose/checkup-type"},
			{Name: "configStatusSourceNode", JSONPath: `{.data.status\.result\.sourceNode}`, matchLabel: "kiagnose/checkup-type"},
		},
	},
	"DataSource.cdi.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "pvcName", JSONPath: `{.spec.source.pvc.name}`},
			{Name: "pvcNamespace", JSONPath: `{.spec.source.pvc.namespace}`},
			{Name: "snapshotName", JSONPath: `{.spec.source.snapshot.name}`},
			{Name: "snapshotNamespace", JSONPath: `{.spec.source.snapshot.namespace}`},
		},
		extractConditions: true,
	},
	"DataVolume.cdi.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "size", JSONPath: `{.spec.storage.resources.requests.storage}`},
			{Name: "snapshotName", JSONPath: `{.spec.source.snapshot.name}`},
			{Name: "snapshotNamespace", JSONPath: `{.spec.source.snapshot.namespace}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
			{Name: "pvcName", JSONPath: `{.spec.source.pvc.name}`},
			{Name: "pvcNamespace", JSONPath: `{.spec.source.pvc.namespace}`},
			{Name: "storageClassName", JSONPath: `{.spec.storage.storageClassName}`},
		},
		extractAnnotations: true,
	},
	"DataImportCron.cdi.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "managedDataSource", JSONPath: `{.spec.managedDataSource}`},
		},
		extractAnnotations: true,
	},
	"Job": {
		properties: []ExtractProperty{
			{Name: "active", JSONPath: `{.status.active}`},
			{Name: "failed", JSONPath: `{.status.failed}`},
		},
	},
	"MigrationPolicy.migrations.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "allowAutoConverge", JSONPath: `{.spec.allowAutoConverge}`},
			{Name: "allowPostCopy", JSONPath: `{.spec.allowPostCopy}`},
			{Name: "bandwidthPerMigration", JSONPath: `{.spec.bandwidthPerMigration}`, DataType: DataTypeBytes},
			{Name: "completionTimeoutPerGiB", JSONPath: `{.spec.completionTimeoutPerGiB}`},
			{Name: "_namespaceSelector", JSONPath: `{.spec.selectors.namespaceSelector}`, DataType: DataTypeMapString},
			{Name: "_virtualMachineInstanceSelector", JSONPath: `{.spec.selectors.virtualMachineInstanceSelector}`, DataType: DataTypeMapString},
		},
		extractAnnotations: true,
	},
	"MultiClusterHub.operator.open-cluster-management.io": {
		extractConditions: true,
	},
	"Namespace": {
		properties: []ExtractProperty{
			{Name: "status", JSONPath: `{.status.phase}`},
		},
	},
	"NetworkAddonsConfig.networkaddonsoperator.network.kubevirt.io": {
		extractConditions: true,
	},
	"NetworkAttachmentDefinition.k8s.cni.cncf.io": {
		extractAnnotations: true,
	},
	"Node": {
		properties: []ExtractProperty{
			{Name: "ipAddress", JSONPath: `{.status.addresses[?(@.type=="InternalIP")].address}`},
			{Name: "memoryAllocatable", JSONPath: `{.status.allocatable.memory}`, DataType: DataTypeBytes},
			{Name: "memoryCapacity", JSONPath: `{.status.capacity.memory}`, DataType: DataTypeBytes},
		},
		extractConditions: true,
	},
	"PersistentVolumeClaim": {
		properties: []ExtractProperty{
			{Name: "requestedStorage", JSONPath: `{.spec.resources.requests.storage}`, DataType: DataTypeBytes},
			{Name: "volumeMode", JSONPath: `{.spec.volumeMode}`},
		},
	},
	"Pod": {
		extractConditions: true,
	},
	"Search.search.open-cluster-management.io": {
		extractConditions: true,
	},
	"StorageClass.storage.k8s.io": {
		properties: []ExtractProperty{
			{Name: "allowVolumeExpansion", JSONPath: `{.allowVolumeExpansion}`},
			{Name: "provisioner", JSONPath: `{.provisioner}`},
			{Name: "reclaimPolicy", JSONPath: `{.reclaimPolicy}`},
			{Name: "volumeBindingMode", JSONPath: `{.volumeBindingMode}`},
		},
	},
	"Subscription.operators.coreos.com": {
		properties: []ExtractProperty{
			{Name: "source", JSONPath: "{.spec.source}"},
			{Name: "package", JSONPath: "{.spec.name}"},
			{Name: "channel", JSONPath: "{.spec.channel}"},
			{Name: "installplan", JSONPath: "{.status.installedCSV}"},
			{Name: "phase", JSONPath: "{.status.state}"},
		},
	},
	"Template.template.openshift.io": {
		properties: []ExtractProperty{
			{Name: "objectVMName", JSONPath: `{.objects[0].metadata.name}`, matchLabel: "template.kubevirt.io/type"},
			{Name: "objectVMArchitecture", JSONPath: `{.objects[0].spec.template.spec.architecture}`, matchLabel: "template.kubevirt.io/type"},
		},
	},
	"ValidatingAdmissionPolicy.admissionregistration.k8s.io": {
		properties: []ExtractProperty{
			{Name: "paramKind_kind", JSONPath: `{.spec.paramKind.kind}`, metadataOnly: true},
			{Name: "paramKind_apiVersion", JSONPath: `{.spec.paramKind.apiVersion}`, metadataOnly: true},
		},
	},
	"VirtualMachine.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "architecture", JSONPath: `{.spec.template.spec.architecture}`},
			{Name: "agentConnected", JSONPath: `{.status.conditions[?(@.type=="AgentConnected")].status}`},
			{Name: "cpu", JSONPath: `{.spec.template.spec.domain.cpu.cores}`},
			{Name: "dataVolumeNames", JSONPath: `{.spec.template.spec.volumes[*].dataVolume.name}`, DataType: DataTypeSlice},
			{Name: "_description", JSONPath: `{.metadata.annotations.description}`},
			{Name: "flavor", JSONPath: `{.spec.template.metadata.annotations.\vm\.kubevirt\.io/flavor}`},
			{Name: "gpuName", JSONPath: `{.spec.template.spec.domain.devices.gpus[*].name}`, DataType: DataTypeSlice},
			{Name: "hostDeviceName", JSONPath: `{.spec.template.spec.domain.devices.hostDevices[*].name}`, DataType: DataTypeSlice},
			{Name: "instancetype", JSONPath: `{.spec.instancetype.name}`},
			{Name: "memory", JSONPath: `{.spec.template.spec.domain.memory.guest}`, DataType: DataTypeBytes},
			{Name: "osName", JSONPath: `{.spec.template.metadata.annotations.\vm\.kubevirt\.io/os}`},
			{Name: "preference", JSONPath: `{.spec.preference.name}`},
			{Name: "pvcClaimNames", JSONPath: `{.spec.template.spec.volumes[*].persistentVolumeClaim.claimName}`, DataType: DataTypeSlice},
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "runStrategy", JSONPath: `{.spec.runStrategy}`},
			{Name: "status", JSONPath: `{.status.printableStatus}`},
			{Name: "_specRunning", JSONPath: `{.spec.running}`},
			{Name: "_specRunStrategy", JSONPath: `{.spec.runStrategy}`},
			{Name: "workload", JSONPath: `{.spec.template.metadata.annotations.\vm\.kubevirt\.io/workload}`},
		},
		edges: []ExtractEdge{
			{Name: "dataVolumeNames", ToKind: "DataVolume", Type: attachedTo},
			{Name: "pvcClaimNames", ToKind: "PersistentVolumeClaim", Type: attachedTo},
		},
		extractConditions: true,
	},
	"VirtualMachineClone.clone.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "targetName", JSONPath: `{.spec.target.name}`},
			{Name: "targetKind", JSONPath: `{.spec.target.kind}`},
			{Name: "sourceName", JSONPath: `{.spec.source.name}`},
			{Name: "sourceKind", JSONPath: `{.spec.source.kind}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
		},
		extractConditions: true,
	},
	"VirtualMachineClusterInstancetype.instancetype.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "cpuGuest", JSONPath: `{.spec.cpu.guest}`},
			{Name: "memoryGuest", JSONPath: `{.spec.memory.guest}`, DataType: DataTypeBytes},
		},
	},
	"VirtualMachineInstance.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "cpu", JSONPath: `{.spec.domain.cpu.cores}`},
			{Name: "cpuSockets", JSONPath: `{.spec.domain.cpu.sockets}`},
			{Name: "cpuThreads", JSONPath: `{.spec.domain.cpu.threads}`},
			{Name: "guestOSInfoID", JSONPath: `{.status.guestOSInfo.id}`},
			{Name: "interfaceName", JSONPath: `{.spec.domain.devices.interfaces[*].name}`, DataType: DataTypeSlice},
			{Name: "_interface", JSONPath: `{.status.interfaces[*]}`},
			{Name: "ipaddress", JSONPath: `{.status.interfaces[0].ipAddress}`},
			{Name: "liveMigratable", JSONPath: `{.status.conditions[?(@.type=='LiveMigratable')].status}`},
			{Name: "memory", JSONPath: `{.spec.domain.memory.guest}`, DataType: DataTypeBytes},
			{Name: "migrationPolicyName", JSONPath: `{.status.migrationState.migrationPolicyName}`},
			{Name: "node", JSONPath: `{.status.nodeName}`},
			{Name: "osVersion", JSONPath: `{.status.guestOSInfo.version}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "startStrategy", JSONPath: `{.spec.startStrategy}`},
			{Name: "vmSize", JSONPath: `{.metadata.labels.\kubevirt\.io/size}`},
		},
	},
	"VirtualMachineInstanceMigration.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "deleted", JSONPath: `{.metadata.deletionTimestamp}`},
			{Name: "endTime", JSONPath: `{.status.migrationState.endTimestamp}`},
			{Name: "migrationPolicyName", JSONPath: `{.status.migrationState.migrationPolicyName}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
			{Name: "sourceNode", JSONPath: `{.status.migrationState.sourceNode}`},
			{Name: "sourcePod", JSONPath: `{.status.migrationState.sourcePod}`},
			{Name: "targetNode", JSONPath: `{.status.migrationState.targetNode}`},
			{Name: "vmiName", JSONPath: `{.spec.vmiName}`},
			{Name: "vmiName", JSONPath: `{.spec.vmiName}`, metadataOnly: true}, // Used to build the edge
		},
		edges: []ExtractEdge{
			{Name: "vmiName", ToKind: "VirtualMachineInstance", Type: migrationOf},
		},
	},
	"VirtualMachineInstancetype.instancetype.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "cpuGuest", JSONPath: `{.spec.cpu.guest}`},
			{Name: "memoryGuest", JSONPath: `{.spec.memory.guest}`, DataType: DataTypeBytes},
		},
	},
	"VirtualMachineSnapshot.snapshot.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "_conditionReadyReason", JSONPath: `{.status.conditions[?(@.type=='Ready')].reason}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
			{Name: "indications", JSONPath: `{.status.indications}`}, // this is an array of strings - will collect array items separated by ;
			{Name: "sourceKind", JSONPath: `{.spec.source.kind}`},
			{Name: "sourceName", JSONPath: `{.spec.source.name}`},
			{Name: "readyToUse", JSONPath: `{.status.readyToUse}`},
		},
		extractConditions: true,
	},
	"VirtualMachineRestore.snapshot.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "_conditionReadyReason", JSONPath: `{.status.conditions[?(@.type=='Ready')].reason}`},
			{Name: "restoreTime", JSONPath: `{.status.restoreTime}`},
			{Name: "complete", JSONPath: `{.status.complete}`},
			{Name: "targetApiGroup", JSONPath: `{.spec.target.apiGroup}`},
			{Name: "targetKind", JSONPath: `{.spec.target.kind}`},
			{Name: "targetName", JSONPath: `{.spec.target.name}`},
			{Name: "virtualMachineSnapshotName", JSONPath: `{.spec.virtualMachineSnapshotName}`},
		},
	},
	"VolumeSnapshot.snapshot.storage.k8s.io": {
		properties: []ExtractProperty{
			{Name: "volumeSnapshotClassName", JSONPath: `{.spec.volumeSnapshotClassName}`},
			{Name: "persistentVolumeClaimName", JSONPath: `{.spec.source.persistentVolumeClaimName}`},
			{Name: "restoreSize", JSONPath: `{.status.restoreSize}`, DataType: DataTypeBytes},
		},
	},
}

// Get the properties to extract from a resource.
func getTransformConfig(group, kind string) (ResourceConfig, bool) {
	transformConfig := defaultTransformConfig

	// FUTURE: We want to create this dynamically by reading from:
	// 	  1. ConfigMap where it can be customized by the users.
	// 	  2. CRD "additionalPrinterColumns" field.

	var val ResourceConfig
	var found bool

	if group == "" { // kubernetes core api resources
		val, found = transformConfig[kind]
	} else {
		val, found = transformConfig[kind+"."+group]
	}

	return val, found
}

// LoadAndMergeConfigurableCollection loads the CollectionConfig resource from the cluster and merges it with defaultTransformConfig
func LoadAndMergeConfigurableCollection() {
	if !config.Cfg.FeatureConfigurableCollection {
		klog.Info("Configurable collection feature is disabled, skipping custom config load")
		return
	}

	klog.Info("Loading configurable collection config from cluster")

	dynamicClient := config.GetDynamicClient()
	// FUTURE: ACM-20047 watch this for changes and update config dynamically
	gvr := schema.GroupVersionResource{
		Group:    "search.open-cluster-management.io",
		Version:  "v1alpha1",
		Resource: "collectionconfigs",
	}

	resource, err := dynamicClient.Resource(gvr).Namespace(config.Cfg.PodNamespace).
		Get(context.Background(), "collection-config", metav1.GetOptions{})

	if err != nil {
		klog.Warningf("Could not load collection-config resource: %v. Using default config only", err)
		return
	}

	klog.Info("Found collection-config resource, merging with default config")

	// get spec field from resource
	spec, specFound, _ := unstructuredNested(resource.Object, "spec")
	if !specFound {
		klog.Warning("No spec found in collection-config resource. Using default config only")
		return
	}

	specMap, ok := spec.(map[string]interface{})
	if !ok {
		klog.Warning("spec is not a map in collection-config resource. Using default config only")
		return
	}

	// FUTURE: ACM-21531
	if collectNamespaces, nsFound, _ := unstructuredNested(specMap, "collectNamespaces"); nsFound {
		if nsMap, ok := collectNamespaces.(map[string]interface{}); ok {
			if _, selectorFound, _ := unstructuredNested(nsMap, "namespaceSelector"); selectorFound {
				klog.V(2).Info("namespaceSelector found in collection-config but not yet implemented. Ignoring.")
			}
		}
	}

	// get collectionRules from spec
	collectionRules, rulesFound, _ := unstructuredNested(specMap, "collectionRules")
	if !rulesFound {
		klog.Info("No collectionRules found in collection-config resource")
		return
	}

	rulesArray, ok := collectionRules.([]interface{})
	if !ok {
		klog.Warning("collectionRules is not an array in collection-config resource. Using default config only")
		return
	}

	// merge each rule from collectionRules with defaultTransformConfig
	for _, rule := range rulesArray {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			klog.Warning("collectionRules item is not a map, skipping")
			continue
		}

		// FUTURE:
		// Only process Include actions
		action, _ := ruleMap["action"].(string)
		if action != "Include" {
			klog.V(2).Infof("Skipping non-Include action: %s", action)
			continue
		}

		// Only process rules that have fields specified
		fields, hasFields := ruleMap["fields"].([]interface{})
		if !hasFields || len(fields) == 0 {
			klog.V(2).Info("Skipping Include action without fields specified")
			continue
		}

		// Extract resourceSelector
		resourceSelector, hasSelectorMap := ruleMap["resourceSelector"].(map[string]interface{})
		if !hasSelectorMap {
			klog.Warning("collectionRules item missing resourceSelector, skipping")
			continue
		}

		apiGroups, _ := resourceSelector["apiGroups"].([]interface{})
		kinds, _ := resourceSelector["kinds"].([]interface{})

		if len(kinds) == 0 {
			klog.Warning("collectionRules item missing kinds in resourceSelector, skipping")
			continue
		}

		// Extract optional flags
		// FUTURE: ACM-30891
		collectAnnotations, _ := ruleMap["collectAnnotations"].(bool)
		// FUTURE: ACM-2071
		collectConditions, _ := ruleMap["collectConditions"].(bool)

		// validation webhook should ensure there's not >1 apiGroup.kind
		// When fields are specified, there should be exactly one kind and one apiGroup
		if len(kinds) != 1 {
			klog.Warningf("Include action with fields must have exactly 1 kind, found %d. Skipping rule.", len(kinds))
			continue
		}

		if len(apiGroups) != 1 {
			klog.Warningf("Include action with fields must have exactly 1 apiGroup, found %d. Skipping rule.", len(apiGroups))
			continue
		}

		// Extract the single kind
		kind, ok := kinds[0].(string)
		if !ok || kind == "" {
			klog.Warning("Kind is not a valid string, skipping rule")
			continue
		}

		// Extract the single apiGroup (empty string "" for core API)
		apiGroup, ok := apiGroups[0].(string)
		if !ok {
			klog.Warning("ApiGroup is not a valid string, skipping rule")
			continue
		}

		resourceKey := kind
		if apiGroup != "" {
			resourceKey = kind + "." + apiGroup
		}

		// get existing key for kind.apiGroup resource
		resourceConfig, exists := defaultTransformConfig[resourceKey]
		if !exists {
			resourceConfig = ResourceConfig{
				properties: []ExtractProperty{},
			}
		}

		// Set annotation and condition flags if specified
		// FUTURE: ACM-30891
		if collectAnnotations {
			klog.V(2).Info("collectAnnotations found in collection-config but not yet implemented. Ignoring.")
			// resourceConfig.extractAnnotations = true
		}
		// FUTURE: ACM-2071
		if collectConditions {
			klog.V(2).Info("collectConditions found in collection-config but not yet implemented. Ignoring.")
			// resourceConfig.extractConditions = true
		}

		// parse and add new fields to resourceConfig
		for _, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := fieldMap["name"].(string)
			jsonPath, _ := fieldMap["jsonPath"].(string)
			dataTypeStr, dataTypeOK := fieldMap["type"].(string)
			//priority, _ := fieldMap["priority"].(string) // FUTURE: use this for additionalPrinterColumns extensions

			if name == "" || jsonPath == "" {
				klog.Warningf("Field missing name or jsonPath for resource %s, skipping", resourceKey)
				continue
			}

			/* TODO: come up with prefix schema before implementation. e.g. The specific configurable collection resource we read from determines the prefix to use
			user defined: user_myResource
			grc  defined: grc_thisPolicyThing
			virt defined: virt_thatVMWhatchamacallit
			*/
			name = "user_" + name

			extractProp := ExtractProperty{
				Name:     name,
				JSONPath: jsonPath,
			}

			if dataTypeOK {
				extractProp.DataType = stringToDataType(dataTypeStr)
			}

			// FUTURE: collision handling with ExtractProperties that already exist in config, the first ExtractProperty
			// that gets parsed and stored in the node properties wins, the duplicate gets skipped
			resourceConfig.properties = append(resourceConfig.properties, extractProp)
			klog.V(2).Infof("Added custom field %s to resource %s", name, resourceKey)
		}

		// Update the config
		defaultTransformConfig[resourceKey] = resourceConfig
		klog.Infof("Merged %d custom fields for resource %s", len(fields), resourceKey)
	}

	klog.Info("Successfully merged configurable collection config")
}

// Helper function to safely access nested fields in unstructured data
func unstructuredNested(obj map[string]interface{}, fields ...string) (interface{}, bool, error) {
	var val interface{} = obj
	for _, field := range fields {
		m, ok := val.(map[string]interface{})
		if !ok {
			return nil, false, nil
		}
		val, ok = m[field]
		if !ok {
			return nil, false, nil
		}
	}
	return val, true, nil
}
