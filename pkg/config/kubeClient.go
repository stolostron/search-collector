// Copyright (c) 2020 Red Hat, Inc.

package config

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// func GetDiscoveryClient() *discovery.DiscoveryClient {
// 	var clientConfig *rest.Config
// 	var clientConfigError error

// 	if Cfg.KubeConfig != "" {
// 		glog.Infof("Creating k8s client using path: %s", Cfg.KubeConfig)
// 		clientConfig, clientConfigError = clientcmd.BuildConfigFromFlags("", Cfg.KubeConfig)
// 	} else {
// 		glog.Info("Creating k8s client using InClusterlientConfig()")
// 		clientConfig, clientConfigError = rest.InClusterConfig()
// 	}

// 	if clientConfigError != nil {
// 		glog.Fatal("Error Constructing Client From Config: ", clientConfigError)
// 	}

// 	discoveryClient, err := discovery.NewDiscoveryClientForConfig(clientConfig)
// 	if err != nil {
// 		glog.Fatal("Cannot Construct Discovery Client From Config: ", err)
// 	}

// 	return discoveryClient
// }

// func GetKubeClient() *kubernetes.Clientset {
// 	clientConfig, _ := clientcmd.BuildConfigFromFlags("", Cfg.KubeConfig)
// 	clientset, err := kubernetes.NewForConfig(clientConfig)
// 	if err != nil {
// 		panic(err.Error())
// 	}

// 	return clientset
// }

func GetDynamicClient() dynamic.Interface {
	clientConfig, _ := clientcmd.BuildConfigFromFlags("", Cfg.KubeConfig)
	clientset, err := dynamic.NewForConfig(clientConfig)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}
