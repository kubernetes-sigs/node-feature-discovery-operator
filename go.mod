module github.com/kubernetes-sigs/node-feature-discovery-operator

go 1.14

require (
	github.com/go-openapi/spec v0.19.3
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/openshift/api v0.0.0-20200116145750-0e2ff1e215dd
	github.com/operator-framework/operator-sdk v0.4.1-0.20190129222657-43d37ce85826
	golang.org/x/text v0.3.3 // indirect
	// Kubernetes 1.17
	k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver v0.17.3 // indirect
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/kube-openapi v0.0.0
	sigs.k8s.io/controller-runtime v0.4.0
)

replace (
	k8s.io/client-go => k8s.io/client-go v0.17.3
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
)
