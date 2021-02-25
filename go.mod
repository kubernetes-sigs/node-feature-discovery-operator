module github.com/kubernetes-sigs/node-feature-discovery-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/custom-resource-status v0.0.0-20210221154447-420d9ecf2a00
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/cri-api v0.20.4
	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.20.4
	sigs.k8s.io/controller-runtime v0.7.0
)
