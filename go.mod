// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

module github.com/stolostron/search-collector

go 1.24.0

toolchain go1.24.4

require (
	github.com/golang/glog v1.0.0
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8
	github.com/kennygrant/sanitize v1.2.4
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/stolostron/governance-policy-propagator v0.0.0-20220125192743-95d49290a318
	github.com/stolostron/multicloud-operators-deployable v1.2.4-1-20220201-2d1add0
	github.com/stolostron/multicloud-operators-placementrule v1.2.4-1-20220311-8eedb3f
	github.com/stretchr/testify v1.10.0
	github.com/tkanos/gonfig v0.0.0-20210106201359-53e13348de2f
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/time v0.9.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.33.2
	k8s.io/apimachinery v0.33.2
	k8s.io/client-go v13.0.0+incompatible
	k8s.io/helm v2.17.0+incompatible
	k8s.io/klog/v2 v2.130.1
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738 // indirect
	open-cluster-management.io/multicloud-operators-channel v0.8.0
	open-cluster-management.io/multicloud-operators-subscription v0.8.0 //Use 2.0 when available
	sigs.k8s.io/application v0.8.3
)

require github.com/stolostron/klusterlet-addon-controller v0.0.0-20221125104750-d4b167d5fae6

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/open-cluster-management/multicloud-operators-placementrule v1.2.4-0-20211122-be034 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stolostron/cluster-lifecycle-api v0.0.0-20220621134646-8b67f2e6afed // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/oauth2 v0.28.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250318190949-c8a335a9a2ff // indirect
	open-cluster-management.io/api v0.8.0 // indirect
	sigs.k8s.io/controller-runtime v0.12.3 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.6.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.33.2
