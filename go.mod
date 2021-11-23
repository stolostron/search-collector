// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

module github.com/open-cluster-management/search-collector

go 1.16

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
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
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v13.0.0+incompatible
	k8s.io/helm v2.16.7+incompatible
	k8s.io/klog/v2 v2.9.0 // indirect
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9 // indirect
	sigs.k8s.io/application v0.8.3
)

replace k8s.io/client-go => k8s.io/client-go v0.21.1
