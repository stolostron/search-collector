// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

module github.com/stolostron/search-collector

go 1.26.0

require (
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8
	github.com/kennygrant/sanitize v1.2.4
	github.com/openshift/api v0.0.0-20260213155647-8fe9fe363807
	github.com/stolostron/governance-policy-propagator v0.0.0-20220125192743-95d49290a318
	github.com/stolostron/multicloud-operators-deployable v1.2.4-1-20220201-2d1add0
	github.com/stretchr/testify v1.11.1
	github.com/tkanos/gonfig v0.0.0-20210106201359-53e13348de2f
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.34.3
	k8s.io/apimachinery v0.34.3
	k8s.io/client-go v13.0.0+incompatible
	k8s.io/helm v2.17.0+incompatible
	k8s.io/klog/v2 v2.130.1
	k8s.io/utils v0.0.0-20260108192941-914a6e750570 // indirect
	open-cluster-management.io/multicloud-operators-channel v0.8.0
	open-cluster-management.io/multicloud-operators-subscription v0.8.0 //Use 2.0 when available
	sigs.k8s.io/application v0.8.3
)

require (
	github.com/gorilla/mux v1.8.1
	github.com/openshift/controller-runtime-common v0.0.0-20260213175913-767fef058eca
	github.com/prometheus/client_golang v1.23.2
	github.com/stolostron/klusterlet-addon-controller v0.0.0-20221125104750-d4b167d5fae6
	github.com/stolostron/search-v2-operator v0.0.0-20260709195409-11944e2ffed6
	sigs.k8s.io/cluster-api v1.10.4
)

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/open-cluster-management/multicloud-operators-placementrule v1.2.4-0-20211122-be034 // indirect
	github.com/openshift/library-go v0.0.0-20260213153706-03f1709971c5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/stolostron/cluster-lifecycle-api v0.0.0-20220621134646-8b67f2e6afed // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/term v0.43.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiextensions-apiserver v0.34.3 // indirect
	k8s.io/apiserver v0.34.3 // indirect
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // indirect
	open-cluster-management.io/api v0.16.0 // indirect
	sigs.k8s.io/controller-runtime v0.22.5 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.2-0.20260122202528-d9cc6641c482 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.34.3
