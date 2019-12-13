package nodefeaturediscovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

type statusCallback func(obj *unstructured.Unstructured) bool

// makeStatusCallback Closure capturing json path and expected status
func makeStatusCallback(obj *unstructured.Unstructured, status interface{}, fields ...string) func(obj *unstructured.Unstructured) bool {
	_status := status
	_fields := fields
	return func(obj *unstructured.Unstructured) bool {
		switch x := _status.(type) {
		case int, int32, int8, int64:

			expected := _status.(int)
			current, found, _ := unstructured.NestedInt64(obj.Object, _fields...)
			if !found {
				log.Info("Not found, ignoring")
				return true
			}
			if current == int64(expected) {
				return true
			}
			return false
		case string:
			expected := _status.(string)
			current, found, err := unstructured.NestedString(obj.Object, _fields...)
			checkNestedFields(found, err)

			if stat := strings.Compare(current, expected); stat == 0 {
				return true
			}
			return false

		default:
			panic(fmt.Errorf("cannot extract type from %T", x))

		}
	}
}

func waitForResource(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery) error {
	if obj.GetKind() == "DaemonSet" {
		return waitForDaemonSet(obj, r)
	}
	if obj.GetKind() == "Pod" {
		return waitForPod(obj, r)
	}
	return nil
}

func waitForPod(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery) error {
	if err := waitForResourceAvailability(obj, r); err != nil {
		return err
	}
	callback := makeStatusCallback(obj, "Succeeded", "status", "phase")
	return waitForResourceFullAvailability(obj, r, callback)
}

func waitForDaemonSet(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery) error {
	if err := waitForResourceAvailability(obj, r); err != nil {
		return err
	}
	callback := makeStatusCallback(obj, 0, "status", "numberUnavailable")
	return waitForResourceFullAvailability(obj, r, callback)
}

// WAIT FOR RESOURCES -- other file?

var (
	retryInterval = time.Second * 5
	timeout       = time.Second * 60
)

func waitForResourceAvailability(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery) error {
	found := obj.DeepCopy()
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, found)
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("Waiting for creation of ", "Namespace", obj.GetNamespace(), "Name", obj.GetName())
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	return err
}

func waitForResourceFullAvailability(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery, callback statusCallback) error {
	found := obj.DeepCopy()
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, found)
		if err != nil {
			log.Error(err, "")
			return false, err
		}
		if callback(found) {
			log.Info("Resource available ", "Namespace", obj.GetNamespace(), "Name", obj.GetName())
			return true, nil
		}
		log.Info("Waiting for availability of ", "Namespace", obj.GetNamespace(), "Name", obj.GetName())
		return false, nil
	})
	return err
}
