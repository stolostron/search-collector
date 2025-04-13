// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

module github.com/stolostron/search-collector

go 1.23.0

require (
	github.com/golang/glog v1.2.4
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8
	github.com/kennygrant/sanitize v1.2.4
	github.com/openshift/api 10a8fa583797
	github.com/stolostron/governance-policy-propagator 7b660d8b8f31
	github.com/stolostron/multicloud-operators-deployable v1.2.4-1-20220201-2d1add0
	github.com/stolostron/multicloud-operators-placementrule v1.2.4-1-20220311-8eedb3f
	github.com/stretchr/testify v1.10.0
	github.com/tkanos/gonfig v0.0.0-20210106201359-53e13348de2f
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.28.2
	k8s.io/apimachinery v0.28.2
	k8s.io/client-go v13.0.0+incompatible
	k8s.io/helm v2.17.0+incompatible
	k8s.io/klog/v2 v2.100.1
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2 // indirect
	open-cluster-management.io/multicloud-operators-channel v0.8.0
	open-cluster-management.io/multicloud-operators-subscription v0.8.0 //Use 2.0 when available
	sigs.k8s.io/application v0.8.3
)

require (
	github.com/gorilla/mux v1.8.1
	github.com/prometheus/client_golang v1.22.0
	github.com/stolostron/klusterlet-addon-controller ace4634835aa
)

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch v5.9.11+incompatible // indirect
	github.com/ghodss/yaml d8423dcdf344 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/open-cluster-management/multicloud-operators-placementrule v1.2.4-1-20220311-8eedb3f // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.63.0 // indirect
	github.com/prometheus/procfs v0.16.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/stolostron/cluster-lifecycle-api a5e71b599118 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/oauth2 v0.28.0 // indirect
	golang.org/x/term v0.27.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230717233707-2695361300d9 // indirect
	open-cluster-management.io/api v0.8.0 // indirect
	sigs.k8s.io/controller-runtime v0.12.3 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.28.2
