package nodefeaturediscovery

import (
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type resourceCallback map[string]func(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery) error

var prefix resourceCallback
var postfix resourceCallback

// SetupCallbacks preassign callbacks for manipulating and handling of resources
func SetupCallbacks() error {

	prefix = make(resourceCallback)
	postfix = make(resourceCallback)

	prefix["prefix-nfd-master"] = prefixNFDmaster
	prefix["prefix-nfd-worker"] = prefixNFDworker
	prefix["prefix-crb-nfd-master"] = prefixCrbNFDmaster
	prefix["prefix-scc-nfd-worker"] = prefixSccNFDworker

	return nil
}

func checkNestedFields(found bool, err error) {
	if !found || err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
}

func prefixResourceCallback(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery) error {

	var ok bool
	todo := ""
	annotations := obj.GetAnnotations()

	if todo, ok = annotations["callback"]; !ok {
		return nil
	}

	if fix, ok := prefix[todo]; ok {
		return fix(obj, r)
	}
	return nil
}

func postfixResourceCallback(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery) error {

	var ok bool
	todo := ""
	annotations := obj.GetAnnotations()
	todo = annotations["callback"]

	if todo, ok = annotations["callback"]; !ok {
		return nil
	}

	if fix, ok := postfix[todo]; ok {
		return fix(obj, r)
	}

	if err := waitForResource(obj, r); err != nil {
		return err
	}

	return nil
}
