// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

module github.com/stolostron/search-collector

go 1.17

require (
	github.com/golang/glog v1.0.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/kennygrant/sanitize v1.2.4
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/stolostron/governance-policy-propagator v0.0.0-20220125192743-95d49290a318
	github.com/stolostron/multicloud-operators-deployable v1.2.4-0-20220114-a630015d
	github.com/stolostron/multicloud-operators-placementrule v1.2.4-0-20220112-8eedb3f
	github.com/stretchr/testify v1.7.0
	github.com/tkanos/gonfig v0.0.0-20210106201359-53e13348de2f
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v13.0.0+incompatible
	k8s.io/helm v2.16.7+incompatible
	k8s.io/klog/v2 v2.9.0
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9 // indirect
	open-cluster-management.io/multicloud-operators-channel v0.5.1-0.20211122200432-da1610291798
	open-cluster-management.io/multicloud-operators-subscription v0.6.0 //Use 2.0 when available
	sigs.k8s.io/application v0.8.3
)

require (
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/evanphx/json-patch v4.11.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.3 // indirect
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7 // indirect
	sigs.k8s.io/controller-runtime v0.9.2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20211215165025-cf75a172585e
	k8s.io/client-go => k8s.io/client-go v0.21.1
)
