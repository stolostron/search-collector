module github.com/open-cluster-management/search-collector

go 1.12

require (
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/aws/aws-sdk-go v1.25.48 // indirect
	github.com/coreos/etcd v3.3.15+incompatible // indirect
	github.com/coreos/go-systemd v0.0.0-20190620071333-e64a0ec8b42a // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/golang/protobuf v1.4.2
	github.com/google/pprof v0.0.0-20190723021845-34ac40c74b70 // indirect
	github.com/gregjones/httpcache v0.0.0-20190203031600-7a902570cb17 // indirect
	github.com/hashicorp/go-version v1.2.0
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/kennygrant/sanitize v1.2.4
	github.com/open-cluster-management/governance-policy-propagator v0.0.0-20200602150427-d0f4af8aba9d
	github.com/open-cluster-management/multicloud-operators-channel v1.0.0
	github.com/open-cluster-management/multicloud-operators-deployable v1.0.0
	github.com/open-cluster-management/multicloud-operators-foundation v1.0.0
	github.com/open-cluster-management/multicloud-operators-placementrule v1.0.0
	github.com/open-cluster-management/multicloud-operators-subscription v1.0.0
	github.com/open-cluster-management/multicloud-operators-subscription-release v0.0.0-20200421184321-05020fc22ab1
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible // indirect
	github.com/tkanos/gonfig v0.0.0-20181112185242-896f3d81fadf
	gonum.org/v1/gonum v0.0.0-20190710053202-4340aa3071a0 // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v13.0.0+incompatible
	k8s.io/helm v2.16.7+incompatible
)

replace (
	github.com/docker/docker => github.com/docker/docker v1.13.1
	// github.com/kubernetes-sigs/application => github.com/kubernetes-sigs/application v0.8.1
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
// sigs.k8s.io/application => sigs.k8s.io/application v0.4.0
)
