// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/stolostron/search-collector/pkg/config"
	rec "github.com/stolostron/search-collector/pkg/reconciler"
	tr "github.com/stolostron/search-collector/pkg/transforms"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

// transformCRD will strip a CRD to be an unstructured object with the following fields. Note that only the stored
// version is in the spec.versions array.
//
//	apiVersion: apiextensions.k8s.io/v1
//	kind: CustomResourceDefinition
//	metadata:
//	  name: ""
//	  generation: 1
//	  resourceVersion: 1
//	spec:
//	  group: ""
//	  versions:
//	    - name: ""
//	      storage: true
//	      additionalPrinterColumns: []
//	status: {}
func transformCRD(obj interface{}) (interface{}, error) {
	typedObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("expected an Unstructured object")
	}

	transformedObj := unstructured.Unstructured{}
	transformedObj.SetAPIVersion(typedObj.GetAPIVersion())
	transformedObj.SetKind(typedObj.GetKind())
	transformedObj.SetName(typedObj.GetName())
	transformedObj.SetGeneration(typedObj.GetGeneration())
	transformedObj.SetResourceVersion(typedObj.GetResourceVersion())

	group, _, err := unstructured.NestedString(typedObj.Object, "spec", "group")
	if err != nil {
		return nil, err
	}

	err = unstructured.SetNestedField(transformedObj.Object, group, "spec", "group")
	if err != nil {
		return nil, err
	}

	status, _, err := unstructured.NestedMap(typedObj.Object, "status")
	if err != nil {
		return nil, err
	}

	err = unstructured.SetNestedField(transformedObj.Object, status, "status")
	if err != nil {
		return nil, err
	}

	versions, _, err := unstructured.NestedSlice(typedObj.Object, "spec", "versions")
	if err != nil {
		return nil, err
	}

	for _, version := range versions {
		versionTyped, ok := version.(map[string]interface{})
		if !ok {
			continue
		}

		if storage, ok := versionTyped["storage"].(bool); !ok || !storage {
			continue
		}

		transformedVersions := []interface{}{
			map[string]interface{}{
				"name":                     versionTyped["name"],
				"storage":                  versionTyped["storage"],
				"additionalPrinterColumns": versionTyped["additionalPrinterColumns"],
			},
		}

		err = unstructured.SetNestedField(transformedObj.Object, transformedVersions, "spec", "versions")
		if err != nil {
			return nil, err
		}
	}

	return &transformedObj, nil
}

// gvrFromCRD parses the input CRD and returns the GVR of the stored version. Note that this
// relies on Kubernetes' enforcement of the CRD name always being in the <plural>.<group> format
// and the plural resource name always being used as the resource name in the GVR.
// See the naming validation here:
// https://github.com/kubernetes/apiextensions-apiserver/blob/v0.31.0/pkg/apis/apiextensions/validation/validation.go#L74
func gvrFromCRD(crd *unstructured.Unstructured) (*schema.GroupVersionResource, error) {
	var versionName string

	versions, _, _ := unstructured.NestedSlice(crd.Object, "spec", "versions")
	for _, version := range versions {
		versionTyped, ok := version.(map[string]interface{})
		if !ok {
			continue
		}

		if storage, ok := versionTyped["storage"].(bool); !ok || !storage {
			continue
		}

		name, ok := versionTyped["name"].(string)
		if !ok {
			continue
		}

		versionName = name

		break
	}

	if versionName == "" {
		return nil, fmt.Errorf("failed to find the stored version name for the CRD: %s", crd.GetName())
	}

	group, _, _ := unstructured.NestedString(crd.Object, "spec", "group")
	if group == "" {
		return nil, fmt.Errorf("failed to find the group of the CRD: %s", crd.GetName())
	}

	// CRDs must have a name in the format of <plural>.<group>
	// https://github.com/kubernetes/apiextensions-apiserver/blob/v0.31.0/pkg/apis/apiextensions/validation/validation.go#L74
	resource := strings.TrimSuffix(crd.GetName(), "."+group)
	if resource == "" {
		return nil, fmt.Errorf("failed to parse the resource name from the CRD: %s", crd.GetName())
	}

	return &schema.GroupVersionResource{
		Group:    group,
		Version:  versionName,
		Resource: resource,
	}, nil
}

// gvrToPrinterColumns is a concurrency safe mapping of the stored version of a CRD represented as a
// schema.GroupVersionResource with values of ExtractProperty slices which represent the additionalPrinterColumns.
type gvrToPrinterColumns struct {
	lock    sync.RWMutex
	mapping map[schema.GroupVersionResource][]tr.ExtractProperty
}

// toLowerCamelCase converts a UTF-8 string to lower camel case such as enforcementAction. Invalid UTF-8 words are
// ignored. The considered separators are spaces (e.g. "enforcement action") and dashes (e.g. enforcement-action).
func toLowerCamelCase(s string) string {
	if s == "" {
		return s
	}

	// If there are no separators, then preserve the string as is.
	if !strings.Contains(s, " ") && !strings.Contains(s, "-") {
		return s
	}

	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	for _, separator := range []string{" ", "-"} {
		parts := strings.Split(s, separator)

		newS := ""

		for i, part := range parts {
			if i != 0 {
				r, size := utf8.DecodeRuneInString(part)
				// RuneError should never be returned because Kubernetes should ensure it's UTF-8 characters, but this is
				// just an extra precaution.
				if r == utf8.RuneError {
					continue
				}

				part = string(unicode.ToUpper(r)) + part[size:]
			}

			newS += part
		}

		s = newS
	}

	return s
}

// set will parse the GVR from the input CRD and set the additional printer columns in the mapping cache.
func (g *gvrToPrinterColumns) set(crd *unstructured.Unstructured) error {
	gvr, err := gvrFromCRD(crd)
	if err != nil {
		return err
	}

	var printerColumns []tr.ExtractProperty

	versions, _, _ := unstructured.NestedSlice(crd.Object, "spec", "versions")
	for _, version := range versions {
		versionTyped, ok := version.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := versionTyped["name"].(string)
		if name != gvr.Version {
			continue
		}

		additionalPrinterColumns, ok := versionTyped["additionalPrinterColumns"].([]interface{})
		if !ok {
			break
		}

		for _, column := range additionalPrinterColumns {
			columnTyped, ok := column.(map[string]interface{})
			if !ok {
				continue
			}

			name, ok := columnTyped["name"].(string)
			if !ok {
				continue
			}

			jsonPath, ok := columnTyped["jsonPath"].(string)
			if !ok {
				continue
			}

			// The additionalPrinterColumns always have a JSON path without curly braces enclosing it, but the
			// ExtractProperty.JSONPath field expects them.
			jsonPath = fmt.Sprintf("{%s}", jsonPath)

			camelCaseName := toLowerCamelCase(name)

			if camelCaseName == "" {
				continue
			}

			printerColumns = append(printerColumns, tr.ExtractProperty{Name: camelCaseName, JSONPath: jsonPath})
		}

		break
	}

	g.lock.Lock()
	g.mapping[*gvr] = printerColumns
	g.lock.Unlock()

	return nil
}

// unset will parse the stored GVR from the input CRD and delete the mapping cache of it.
func (g *gvrToPrinterColumns) unset(crd *unstructured.Unstructured) error {
	gvr, err := gvrFromCRD(crd)
	if err != nil {
		return err
	}

	g.lock.Lock()
	delete(g.mapping, *gvr)
	g.lock.Unlock()

	return nil
}

// get returns the entries in the additionalPrintedColumns array in the CRD. This uses a cache that must have been
// populated with set.
func (g *gvrToPrinterColumns) get(gvr schema.GroupVersionResource) []tr.ExtractProperty {
	g.lock.RLock()
	result := g.mapping[gvr]
	g.lock.RUnlock()

	return result
}

// isCRDEstablished determines if the CRD has the condition of Established set to True.
func isCRDEstablished(crd *unstructured.Unstructured) bool {
	conditions, _, _ := unstructured.NestedSlice(crd.Object, "status", "conditions")

	for _, condition := range conditions {
		conditionTyped, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		conditionType, ok := conditionTyped["type"].(string)
		if !ok || conditionType != "Established" {
			continue
		}

		conditionStatus, ok := conditionTyped["status"].(string)

		return ok && conditionStatus == "True"
	}

	return false
}

// getCRDInformer returns a started and synced
func getCRDInformer(
	ctx context.Context, gvrToColumns *gvrToPrinterColumns, syncInformersQueue *workqueue.Type,
) (dynamicinformer.DynamicSharedInformerFactory, error) {
	klog.Info("Starting the CRD informer")

	dynSharedInformer := dynamicinformer.NewDynamicSharedInformerFactory(config.GetDynamicClient(), 0)

	crdInformer := dynSharedInformer.ForResource(crdGVR)
	crdInformer.Lister()

	err := crdInformer.Informer().SetTransform(transformCRD)
	if err != nil {
		return nil, err
	}

	// The event handlers just add an empty struct to the syncInformersQueue when any CRD is created, updated, or
	// deleted. Using the empty struct deduplicates the requests so that when multiple CRD updates occur while
	// syncInformers is running, it'll only trigger one additional syncInformers run after the previous completes.
	_, err = crdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			objTyped, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}

			err := gvrToColumns.set(objTyped)
			if err != nil {
				klog.Errorf(
					"Failed to parse the additionalPrinterColumns from the CRD (%s): %v", objTyped.GetName(), err,
				)
			}

			if isCRDEstablished(objTyped) {
				syncInformersQueue.Add(struct{}{})
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newObjTyped, ok := newObj.(*unstructured.Unstructured)
			if !ok {
				return
			}

			if !isCRDEstablished(newObjTyped) {
				return
			}

			err := gvrToColumns.set(newObjTyped)
			if err != nil {
				klog.Errorf(
					"Failed to parse the additionalPrinterColumns from the CRD (%s): %v", newObjTyped.GetName(), err,
				)
			}

			syncInformersQueue.Add(struct{}{})
		},
		DeleteFunc: func(obj interface{}) {
			objTyped, ok := obj.(*unstructured.Unstructured)
			if !ok {
				unknownStateObj, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					return
				}

				objTyped, ok = unknownStateObj.Obj.(*unstructured.Unstructured)
				if !ok {
					return
				}
			}

			err := gvrToColumns.unset(objTyped)
			if err != nil {
				klog.Errorf(
					"Failed to parse the additionalPrinterColumns from the CRD (%s): %v", objTyped.GetName(), err,
				)
			}

			syncInformersQueue.Add(struct{}{})
		},
	})
	if err != nil {
		return nil, err
	}

	dynSharedInformer.Start(ctx.Done())

	klog.Info("Waiting for the CRD informer to sync")

	// Waiting for the CRD informer to sync means that the event handlers have all run for the results from the initial
	// listing of all CRDs. This allows deduplicating the list requests to a single item in the queue.
	if !cache.WaitForCacheSync(ctx.Done(), crdInformer.Informer().HasSynced) {
		return nil, errors.New("timed out waiting for the CRD informer to sync")
	}

	// A bit of a hack to drain the queue from the informer performing the list query. We only really care about
	// events after the initial sync.
	if syncInformersQueue.Len() != 0 {
		item, _ := syncInformersQueue.Get()
		syncInformersQueue.Done(item)
	}

	klog.Info("The CRD informer has started")

	return dynSharedInformer, nil
}

// Start and manages informers for resources in the cluster.
func RunInformers(
	ctx context.Context,
	initialized chan interface{},
	upsertTransformer tr.Transformer,
	reconciler *rec.Reconciler,
) (err error) {
	var wasInitialized bool

	gvrToColumns := gvrToPrinterColumns{mapping: map[schema.GroupVersionResource][]tr.ExtractProperty{}}

	defer func() {
		// Always close the initialized channel if it was not already closed so the caller does not wait forever.
		if !wasInitialized {
			close(initialized)
		}
	}()

	syncInformersQueue := workqueue.NewTypedWithConfig(workqueue.QueueConfig{})
	defer syncInformersQueue.ShutDown()

	dynSharedInformer, err := getCRDInformer(ctx, &gvrToColumns, syncInformersQueue)
	if err != nil {
		return err
	}

	// Get kubernetes client for discovering resource types
	discoveryClient := config.GetDiscoveryClient()

	// We keep each of the informer's stopper channel in a map, so we can stop them if the resource is no longer valid.
	stoppers := make(map[schema.GroupVersionResource]context.CancelFunc)

	// These functions return handler functions, which are then used in creation of the informers.
	createInformAddHandler := func(gvr schema.GroupVersionResource) func(interface{}) {
		return func(obj interface{}) {
			resource := obj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:                     time.Now().Unix(),
				Operation:                tr.Create,
				Resource:                 resource,
				ResourceString:           gvr.Resource,
				AdditionalPrinterColumns: gvrToColumns.get(gvr),
			}
			upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
		}
	}

	createInformUpdateHandler := func(gvr schema.GroupVersionResource) func(interface{}, interface{}) {
		return func(oldObj, newObj interface{}) {
			resource := newObj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:                     time.Now().Unix(),
				Operation:                tr.Update,
				Resource:                 resource,
				ResourceString:           gvr.Resource,
				AdditionalPrinterColumns: gvrToColumns.get(gvr),
			}
			upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
		}
	}

	informDeleteHandler := func(obj interface{}) {
		resource := obj.(*unstructured.Unstructured)
		// We don't actually have anything to transform in the case of a deletion, so we manually construct the NodeEvent
		ne := tr.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: tr.Delete,
			Node: tr.Node{
				UID: strings.Join([]string{config.Cfg.ClusterName, string(resource.GetUID())}, "/"),
			},
		}
		reconciler.Input <- ne
	}

	// Initialize the informers
	syncInformers(ctx, *discoveryClient, stoppers, createInformAddHandler, createInformUpdateHandler, informDeleteHandler)

	// Close the initialized channel so that we can start the sender.
	wasInitialized = true
	close(initialized)

	lastSynced := time.Now()
	minBetweenSyncs := 5 * time.Second

	// Keep the informers synchronized when CRDs are added or deleted in the cluster.
	for {
		select {
		case <-ctx.Done():
			// The parent context canceled, so all the informers's child contexts will also be canceled, so no
			// explicit clean up is needed. Ideally, this would wait for all the informers to have fully stopped before
			// returning, but that state is not available here.
			klog.Info("Waiting for the CRD informer to shutdown")

			// The informer is already shutting down since the parent context was canceled, but this call to Shutdown
			// blocks until all of its goroutines have stopped.
			dynSharedInformer.Shutdown()

			return
		default:

		}

		syncRequest, shutdown := syncInformersQueue.Get()
		if shutdown {
			return
		}

		// Add up to a 5 second delay to account for things such as a new operator adding multiple CRDs.
		sinceLastSync := time.Since(lastSynced)

		if sinceLastSync < minBetweenSyncs {
			time.Sleep(minBetweenSyncs - sinceLastSync)
		}

		syncInformers(
			ctx, *discoveryClient, stoppers, createInformAddHandler, createInformUpdateHandler, informDeleteHandler,
		)

		lastSynced = time.Now()

		syncInformersQueue.Done(syncRequest)
	}
}

// Start or stop informers to match the resources (CRDs) available in the cluster.
func syncInformers(
	ctx context.Context,
	client discovery.DiscoveryClient,
	stoppers map[schema.GroupVersionResource]context.CancelFunc,
	createInformerAddHandler func(schema.GroupVersionResource) func(interface{}),
	createInformerUpdateHandler func(schema.GroupVersionResource) func(interface{}, interface{}),
	informerDeleteHandler func(obj interface{}),
) {
	klog.V(2).Infof("Synchronizing informers. Informers running: %d", len(stoppers))

	gvrList, err := SupportedResources(client)
	if err != nil {
		klog.Error("Failed to get complete list of supported resources: ", err)
	}

	// Sometimes a partial list will be returned even if there is an error.
	// This could happen during install when a CRD hasn't fully initialized.
	if gvrList != nil {
		// Loop through the previous list of resources. If we find the entry in the new list we delete it so
		// that we don't end up with 2 informers. If we don't find it, we stop the informer that's currently
		// running because the resource no longer exists (or no longer supports watch).
		for gvr, stopper := range stoppers {
			// If this still exists in the new list, delete it from there as we don't want to recreate an informer
			if _, ok := gvrList[gvr]; ok {
				delete(gvrList, gvr)
				continue
			} else { // if it's in the old and NOT in the new, stop the informer
				klog.V(2).Infof("Stopping informer: %s", gvr.String())
				stopper()
				delete(stoppers, gvr)
			}
		}
		// Now, loop through the new list, which after the above deletions, contains only stuff that needs to
		// have a new informer created for it.
		for gvr := range gvrList {
			klog.V(2).Infof("Starting informer: %s", gvr.String())
			// Using our custom informer.
			informer, _ := InformerForResource(gvr)

			// Set up handler to pass this informer's resources into transformer
			informer.AddFunc = createInformerAddHandler(gvr)
			informer.UpdateFunc = createInformerUpdateHandler(gvr)
			informer.DeleteFunc = informerDeleteHandler

			informerCtx, informerCancel := context.WithCancel(ctx)
			stoppers[gvr] = informerCancel
			go informer.Run(informerCtx)
			// This wait serializes the informer initialization. It is needed to avoid a
			// spike in memory when the collector starts.
			informer.WaitUntilInitialized(time.Duration(10) * time.Second) // Times out after 10 seconds.
		}
		klog.V(2).Info("Done synchronizing informers. Informers running: ", len(stoppers))
	}
}
