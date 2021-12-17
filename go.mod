// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

module github.com/open-cluster-management/search-collector

go 1.17

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/kennygrant/sanitize v1.2.4
	github.com/kr/text v0.2.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/onsi/gomega v1.10.4 // indirect
	github.com/open-cluster-management/governance-policy-propagator v0.0.0-20200602150427-d0f4af8aba9d
	github.com/open-cluster-management/multicloud-operators-channel v1.0.1-0.20200930214554-fa55cf642642
	github.com/open-cluster-management/multicloud-operators-deployable v0.0.0-20200925154205-fc4ec3e30a4d
	github.com/open-cluster-management/multicloud-operators-foundation v0.0.0-20200629084830-3965fdd47134
	github.com/open-cluster-management/multicloud-operators-placementrule v1.0.1-2020-06-08-14-28-27.0.20200927053545-8e131abcf13c
	github.com/open-cluster-management/multicloud-operators-subscription v1.0.0-2020-05-12-21-17-19.0.20201009005738-cbe273a045ab
	github.com/open-cluster-management/multicloud-operators-subscription-release v1.0.1-2020-06-08-14-28-27.0.20200819124024-818f01d780ff //Use 2.0 when available
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/stretchr/testify v1.7.0
	github.com/tkanos/gonfig v0.0.0-20181112185242-896f3d81fadf
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v13.0.0+incompatible
	k8s.io/helm v2.16.7+incompatible
	sigs.k8s.io/application v0.8.3
)

require (
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-openapi/jsonpointer v0.19.3 // indirect
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
	golang.org/x/term v0.0.0-20201117132131-f5c789dd3221 // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/protobuf v1.23.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
	k8s.io/apiextensions-apiserver v0.18.6 // indirect
	k8s.io/cluster-registry v0.0.6 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6 // indirect
	k8s.io/utils v0.0.0-20200603063816-c1c6865ac451 // indirect
	sigs.k8s.io/controller-runtime v0.6.2 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/buger/jsonparser => github.com/buger/jsonparser v1.1.1
	github.com/deislabs/oras => github.com/deislabs/oras v0.11.1
	github.com/docker/docker => github.com/docker/docker v1.13.1
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200527184302-a843dc3262a0
	github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.8
	golang.org/x/text => golang.org/x/text v0.3.5
	k8s.io/api => k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.4
	k8s.io/apiserver => k8s.io/apiserver v0.17.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.4
	k8s.io/client-go => k8s.io/client-go v0.17.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.4
	k8s.io/code-generator => k8s.io/code-generator v0.17.4
	k8s.io/component-base => k8s.io/component-base v0.17.4
	k8s.io/cri-api => k8s.io/cri-api v0.17.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.4
	k8s.io/kubectl => k8s.io/kubectl v0.17.4
	k8s.io/kubelet => k8s.io/kubelet v0.17.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.4
	k8s.io/metrics => k8s.io/metrics v0.17.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.4
)
