// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package config

import (
	"sync"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var mutex sync.Mutex
var dynamicClient dynamic.Interface

// Get the kubernetes dynamic client.
func GetDynamicClient() dynamic.Interface {
	mutex.Lock()
	defer mutex.Unlock()
	if dynamicClient != nil {
		return dynamicClient
	}
	newDynamicClient, err := dynamic.NewForConfig(GetKubeConfig())
	if err != nil {
		klog.Fatal("Cannot Construct Dynamic Client ", err)
	}
	dynamicClient = newDynamicClient

	return dynamicClient
}

func GetKubeConfig() *rest.Config {
	var clientConfig *rest.Config
	var clientConfigError error

	if Cfg.KubeConfig != "" {
		klog.Infof("Creating k8s client using KubeConfig at: %s", Cfg.KubeConfig)
		clientConfig, clientConfigError = clientcmd.BuildConfigFromFlags("", Cfg.KubeConfig)
	} else {
		klog.V(2).Info("Creating k8s client using InClusterClientConfig()")
		clientConfig, clientConfigError = rest.InClusterConfig()
	}

	if clientConfigError != nil {
		klog.Fatal("Error getting Kube Config: ", clientConfigError)
	}

	return clientConfig
}

// Get kubernetes client for discovering resource types.
func GetDiscoveryClient() *discovery.DiscoveryClient {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(GetKubeConfig())
	if err != nil {
		klog.Fatal("Cannot Construct Discovery Client From Config: ", err)
	}
	return discoveryClient
}

func GetKubeClient(config *rest.Config) *kubernetes.Clientset {
	var kubeClient *kubernetes.Clientset
	var err error
	if config != nil {
		kubeClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			klog.Fatal("Cannot Construct Kube Client from Config: ", err)
		}
	} else {
		klog.Error("Cannot Construct Kube Client as input Config is nil")
	}
	return kubeClient
}
