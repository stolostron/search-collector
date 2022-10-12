/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.

Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	inform "github.com/stolostron/search-collector/pkg/informer"
	lease "github.com/stolostron/search-collector/pkg/lease"
	rec "github.com/stolostron/search-collector/pkg/reconciler"
	tr "github.com/stolostron/search-collector/pkg/transforms"

	"github.com/golang/glog"
	"github.com/stolostron/search-collector/pkg/send"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	AddonName            = "search-collector"
	LeaseDurationSeconds = 60
)

func main() {
	// init logs
	flag.Parse()
	// Glog by default logs to a file. Change it so that by default it all goes to stderr. (no option for stdout).
	err := flag.Lookup("logtostderr").Value.Set("true")
	if err != nil {
		fmt.Println("Error setting default flag:", err) // Uses fmt.Println in case something is wrong with glog args
		os.Exit(1)
	}
	defer glog.Flush() // This should ensure that everything makes it out on to the console if the program crashes.

	// determine number of CPUs available.
	// We make that many goroutines for transformation and reconciliation,
	// so that we take maximum advantage of whatever hardware we're on
	numThreads := runtime.NumCPU()

	glog.Info("Starting Search Collector")
	if commit, ok := os.LookupEnv("VCS_REF"); ok {
		glog.Info("Built from git commit: ", commit)
	}

	if !config.Cfg.DeployedInHub {
		leaseReconciler := lease.LeaseReconciler{
			HubKubeClient:        config.GetKubeClient(config.Cfg.AggregatorConfig),
			LocalKubeClient:      config.GetKubeClient(config.GetKubeConfig()),
			LeaseName:            AddonName,
			ClusterName:          config.Cfg.ClusterName,
			LeaseDurationSeconds: int32(LeaseDurationSeconds),
		}
		glog.Info("Create/Update lease for search")
		go wait.Forever(leaseReconciler.Reconcile, time.Duration(leaseReconciler.LeaseDurationSeconds)*time.Second)
	}

	// Create input channel
	transformChannel := make(chan *tr.Event)

	// Create transformers
	upsertTransformer := tr.NewTransformer(transformChannel, make(chan tr.NodeEvent), numThreads)

	// Init reconciler
	reconciler := rec.NewReconciler()
	reconciler.Input = upsertTransformer.Output

	// Create Sender, attached to transformer
	sender := send.NewSender(reconciler, config.Cfg.AggregatorURL, config.Cfg.ClusterName)

	// Get kubernetes client for discovering resource types
	discoveryClient := config.GetDiscoveryClient()

	// These functions return handler functions, which are then used in creation of the informers.
	createInformerAddHandler := func(resourceName string) func(interface{}) {
		return func(obj interface{}) {
			resource := obj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:           time.Now().Unix(),
				Operation:      tr.Create,
				Resource:       resource,
				ResourceString: resourceName,
			}
			upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
		}
	}

	createInformerUpdateHandler := func(resourceName string) func(interface{}, interface{}) {
		return func(oldObj, newObj interface{}) {
			resource := newObj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:           time.Now().Unix(),
				Operation:      tr.Update,
				Resource:       resource,
				ResourceString: resourceName,
			}
			upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
		}
	}

	informerDeleteHandler := func(obj interface{}) {
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

	informersStarted := false

	// Start a routine to keep our informers up to date.
	go func() {
		// We keep each of the informer's stopper channel in a map, so we can stop them if the resource is no longer valid.
		stoppers := make(map[schema.GroupVersionResource]chan struct{})
		for {
			gvrList, err := inform.SupportedResources(discoveryClient)
			if err != nil {
				glog.Error("Failed to get complete list of supported resources: ", err)
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
						glog.V(2).Infof("Resource %s no longer exists or no longer supports watch, stopping its informer\n", gvr.String())
						close(stopper)
						delete(stoppers, gvr)
					}
				}
				// Now, loop through the new list, which after the above deletions, contains only stuff that needs to
				// have a new informer created for it.
				for gvr := range gvrList {
					fmt.Println("Found new resource %s, creating informer\n", gvr.String())
					if gvr.Resource == "configmaps" {
						fmt.Println("FOUND CONFIGMAP RESOURCE", gvr.String())
					}
					glog.V(2).Infof("Found new resource %s, creating informer\n", gvr.String())
					// Using our custom informer.
					informer, _ := inform.InformerForResource(gvr)

					// Set up handler to pass this informer's resources into transformer
					informer.AddFunc = createInformerAddHandler(gvr.Resource)
					informer.UpdateFunc = createInformerUpdateHandler(gvr.Resource)
					informer.DeleteFunc = informerDeleteHandler

					stopper := make(chan struct{})
					stoppers[gvr] = stopper
					go informer.Run(stopper)
					informer.WaitUntilInitialized(time.Duration(10) * time.Second) // Times out after 10 seconds.
				}
				glog.V(2).Info("Total informers running: ", len(stoppers))
				informersStarted = true
			}

			time.Sleep(time.Duration(config.Cfg.RediscoverRateMS) * time.Millisecond)
		}
	}()

	glog.Info("Waiting for informers to load initial state...")
	for !informersStarted {
		time.Sleep(time.Duration(100) * time.Millisecond)
	}

	glog.Info("Starting the sender.")
	sender.StartSendLoop()
}
