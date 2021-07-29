module github.com/operator/cleanup-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/google/martian v2.1.0+incompatible
	github.com/netapp/trident v0.0.0-20210621162846-dfe11dbe7c83 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/local-storage-operator v0.0.0-20210429014248-65aca4d5aeae
	github.com/operator-framework/api v0.8.0
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/oauth2 v0.0.0-20210413134643-5e61552d6c78 // indirect
	k8s.io/api v0.21.0
	k8s.io/apiextensions-apiserver v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	k8s.io/api => k8s.io/api v0.20.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.6
	k8s.io/apiserver => k8s.io/apiserver v0.20.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.6
	k8s.io/client-go => k8s.io/client-go v0.20.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.6
	k8s.io/code-generator => k8s.io/code-generator v0.20.6
	k8s.io/component-base => k8s.io/component-base v0.20.6
	k8s.io/cri-api => k8s.io/cri-api v0.20.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.6
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.6
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.6
	k8s.io/kubectl => k8s.io/kubectl v0.20.6
	k8s.io/kubelet => k8s.io/kubelet v0.20.6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.6
	k8s.io/metrics => k8s.io/metrics v0.20.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.6
)
