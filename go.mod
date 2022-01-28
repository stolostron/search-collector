// Copyright (c) 2020 Red Hat, Inc.

module github.com/stolostron/search-collector

go 1.15

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/kennygrant/sanitize v1.2.4
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/stolostron/governance-policy-propagator v0.0.0-20220115035716-8b8a6d376df5
	github.com/stolostron/multicloud-operators-deployable v1.2.2-0-20220118-ad940ac1
	github.com/stolostron/multicloud-operators-placementrule v1.2.2-0-20220115-4218674
	github.com/tkanos/gonfig v0.0.0-20181112185242-896f3d81fadf
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/helm v2.16.7+incompatible
	open-cluster-management.io/multicloud-operators-channel v0.6.0
	open-cluster-management.io/multicloud-operators-subscription v0.6.0
	sigs.k8s.io/application v0.8.3
)

replace (
	github.com/buger/jsonparser => github.com/buger/jsonparser v1.1.1
	github.com/docker/docker => github.com/docker/docker v1.13.1
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200527184302-a843dc3262a0
	github.com/stolostron/governance-policy-propagator => github.com/stolostron/governance-policy-propagator v0.0.0-20220115035716-8b8a6d376df5
	golang.org/x/text => golang.org/x/text v0.3.5
	k8s.io/client-go => k8s.io/client-go v0.21.3
)
