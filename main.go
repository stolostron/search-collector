// Copyright Contributors to the Open Cluster Management project

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-collector/pkg/informer"
	lease "github.com/stolostron/search-collector/pkg/lease"
	"github.com/stolostron/search-collector/pkg/mq"
	rec "github.com/stolostron/search-collector/pkg/reconciler"
	tr "github.com/stolostron/search-collector/pkg/transforms"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
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
	reconciler.Input = make(chan tr.NodeEvent) // upsertTransformer.Output
	// Create Sender, attached to transformer
	// sender := send.NewSender(reconciler, config.Cfg.AggregatorURL, config.Cfg.ClusterName)

	go mq.MQReconciler(upsertTransformer.Output)

	informersInitialized := make(chan interface{})
	// Start a routine to keep our informers up to date.
	go informer.RunInformers(informersInitialized, upsertTransformer, reconciler)

	// Wait here until informers have collected the full state of the cluster.
	// The initial payload must have the complete state to avoid unecessary deletion
	// and recreate of existing rows in the database during the resync.
	glog.Info("Waiting for informers to load initial state.")
	<-informersInitialized


	wait.Forever(func() {
		klog.Info("Done")
	}, time.Hour)
	glog.Info("Starting the sender.")
	// sender.StartSendLoop()
}
