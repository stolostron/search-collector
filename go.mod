// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

module github.com/open-cluster-management/search-collector

go 1.16

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/golang/protobuf v1.5.2
	github.com/kennygrant/sanitize v1.2.4
	github.com/open-cluster-management/governance-policy-propagator v0.0.0-20211012174109-95c3b77cce09
	github.com/open-cluster-management/multicloud-operators-channel v1.0.1-0.20200930214554-fa55cf642642
	github.com/open-cluster-management/multicloud-operators-deployable v0.0.0-20200925154205-fc4ec3e30a4d
	github.com/open-cluster-management/multicloud-operators-placementrule v1.2.4-0-20210816-699e5
	github.com/open-cluster-management/multicloud-operators-subscription v1.0.0-2020-05-12-21-17-19.0.20201009005738-cbe273a045ab
	github.com/open-cluster-management/multicloud-operators-subscription-release v1.0.1-2020-06-08-14-28-27.0.20200819124024-818f01d780ff //Use 2.0 when available
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/stretchr/testify v1.7.0
	github.com/tkanos/gonfig v0.0.0-20181112185242-896f3d81fadf
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v13.0.0+incompatible
	k8s.io/helm v2.16.7+incompatible
	k8s.io/klog v1.0.0
	open-cluster-management.io/addon-framework v0.1.0
	sigs.k8s.io/application v0.8.3
	sigs.k8s.io/controller-runtime v0.9.2
)

replace (
	github.com/docker/docker => github.com/docker/docker v1.13.1
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200527184302-a843dc3262a0
	golang.org/x/text => golang.org/x/text v0.3.5
	k8s.io/api => k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.1
	k8s.io/apiserver => k8s.io/apiserver v0.21.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.1
	k8s.io/client-go => k8s.io/client-go v0.21.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.1
	k8s.io/code-generator => k8s.io/code-generator v0.21.1
	k8s.io/component-base => k8s.io/component-base v0.21.1
	k8s.io/cri-api => k8s.io/cri-api v0.21.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.1
	k8s.io/kubectl => k8s.io/kubectl v0.21.1
	k8s.io/kubelet => k8s.io/kubelet v0.21.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.1
	k8s.io/metrics => k8s.io/metrics v0.21.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.1
	open-cluster-management.io/api => open-cluster-management.io/api v0.5.0
)
