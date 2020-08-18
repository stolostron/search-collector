// Copyright (c) 2020 Red Hat, Inc.

package config

import (
	"github.com/golang/glog"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetDynamicClient() dynamic.Interface {
	clientset, err := dynamic.NewForConfig(GetKubeConfig())
	if err != nil {
		glog.Fatal("Cannot Construct Dynamic Client ", err)
	}

	return clientset
}

func GetKubeConfig() *rest.Config {
	var clientConfig *rest.Config
	var clientConfigError error

	if Cfg.KubeConfig != "" {
		glog.Infof("Creating k8s client using path: %s", Cfg.KubeConfig)
		clientConfig, clientConfigError = clientcmd.BuildConfigFromFlags("", Cfg.KubeConfig)
	} else {
		glog.Info("Creating k8s client using InClusterlientConfig()")
		clientConfig, clientConfigError = rest.InClusterConfig()
	}

	if clientConfigError != nil {
		glog.Fatal("Error getting Kube Config: ", clientConfigError)
	}

	return clientConfig
}

// func GetDiscoveryClient() *discovery.DiscoveryClient {
// 	discoveryClient, err := discovery.NewDiscoveryClientForConfig(GetKubeConfig())
// 	if err != nil {
// 		glog.Fatal("Cannot Construct Discovery Client From Config: ", err)
// 	}
// 	return discoveryClient
// }

// func GetKubeClient() *kubernetes.Clientset {
// 	clientConfig, _ := clientcmd.BuildConfigFromFlags("", getKubeConfig())
// 	clientset, err := kubernetes.NewForConfig(clientConfig)
// 	if err != nil {
// 		panic(err.Error())
// 	}
// 	return clientset
// }
