// Copyright Contributors to the Open Cluster Management project

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-collector/pkg/informer"
	lease "github.com/stolostron/search-collector/pkg/lease"
	rec "github.com/stolostron/search-collector/pkg/reconciler"
	"github.com/stolostron/search-collector/pkg/server"
	tr "github.com/stolostron/search-collector/pkg/transforms"
	"k8s.io/klog/v2"

	"github.com/stolostron/search-collector/pkg/send"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	AddonName            = "search-collector"
	LeaseDurationSeconds = 60
)

// getMainContext returns a context that is canceled on SIGINT or SIGTERM signals. If a second signal is received,
// it exits directly.
// This was inspired by controller-runtime.
func getMainContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		cancel()
		<-c
		os.Exit(1) // Second signal. Exit directly.
	}()

	return ctx
}

func main() {
	// Initialize the logger.
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()
	klog.Info("Starting search-collector")

	// determine number of CPUs available.
	// We make that many goroutines for transformation and reconciliation,
	// so that we take maximum advantage of whatever hardware we're on
	numThreads := runtime.NumCPU()

	if commit, ok := os.LookupEnv("VCS_REF"); ok {
		klog.Info("Built from git commit: ", commit)
	}

	config.InitConfig()

	if !config.Cfg.DeployedInHub {
		leaseReconciler := lease.LeaseReconciler{
			HubKubeClient:        config.GetKubeClient(config.Cfg.AggregatorConfig),
			LocalKubeClient:      config.GetKubeClient(config.GetKubeConfig()),
			LeaseName:            AddonName,
			ClusterName:          config.Cfg.ClusterName,
			LeaseDurationSeconds: int32(LeaseDurationSeconds),
		}
		klog.Info("Create/Update lease for search")
		go wait.Forever(leaseReconciler.Reconcile, time.Duration(leaseReconciler.LeaseDurationSeconds)*time.Second)
	}

	// Start metrics server to serve Prometheus metrics
	go server.StartAndListen()

	// Create input channel
	transformChannel := make(chan *tr.Event)

	// Create transformers
	upsertTransformer := tr.NewTransformer(transformChannel, make(chan tr.NodeEvent), numThreads)

	// Init reconciler
	reconciler := rec.NewReconciler()
	reconciler.Input = upsertTransformer.Output

	// Create Sender, attached to transformer
	sender := send.NewSender(reconciler, config.Cfg.AggregatorURL, config.Cfg.ClusterName)

	informersInitialized := make(chan interface{})

	mainCtx := getMainContext()

	wg := sync.WaitGroup{}
	wg.Add(1)

	// Start a routine to keep our informers up to date.
	go func() {
		err := informer.RunInformers(mainCtx, informersInitialized, upsertTransformer, reconciler)
		if err != nil {
			klog.Errorf("Failed to run the informers: %v", err)

			os.Exit(1)
		}

		wg.Done()
	}()

	// Wait here until informers have collected the full state of the cluster.
	// The initial payload must have the complete state to avoid unecessary deletion
	// and recreate of existing rows in the database during the resync.
	klog.Info("Waiting for informers to load initial state.")
	<-informersInitialized

	klog.Info("Starting the sender.")
	wg.Add(1)

	go func() {
		sender.StartSendLoop(mainCtx)
		wg.Done()
	}()

	wg.Wait()
}
