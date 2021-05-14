/*
Copyright 2020-2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nfdv1 "github.com/kubernetes-sigs/node-feature-discovery-operator/api/v1"
)

var log = logf.Log.WithName("controller_nodefeaturediscovery")

var nfd NFD

const finalizer = "foreground-deletion"

// NodeFeatureDiscoveryReconciler reconciles a NodeFeatureDiscovery object
type NodeFeatureDiscoveryReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	AssetsDir string
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeFeatureDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// we want to initate reconcile loop only on spec change of the object
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return validateUpdateEvent(&e)
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nfdv1.NodeFeatureDiscovery{}).
		Owns(&appsv1.DaemonSet{}, builder.WithPredicates(p)).
		Owns(&corev1.Service{}, builder.WithPredicates(p)).
		Owns(&corev1.ServiceAccount{}, builder.WithPredicates(p)).
		Owns(&corev1.Pod{}, builder.WithPredicates(p)).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(p)).
		Complete(r)
}

func validateUpdateEvent(e *event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		klog.Error("Update event has no old runtime object to update")
		return false
	}
	if e.ObjectNew == nil {
		klog.Error("Update event has no new runtime object for update")
		return false
	}

	return true
}

// +kubebuilder:rbac:groups=nfd.kubernetes.io,resources=nodefeaturediscoveries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nfd.kubernetes.io,resources=nodefeaturediscoveries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nfd.kubernetes.io,resources=nodefeaturediscoveries/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=imagestreams/layers,verbs=get
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;update;
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=storage.k8s.io,resources=csinodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=csidrivers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NodeFeatureDiscoveryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("nodefeaturediscovery", req.NamespacedName)

	// Fetch the NodeFeatureDiscovery instance
	r.Log.Info("Fetch the NodeFeatureDiscovery instance")
	instance := &nfdv1.NodeFeatureDiscovery{}
	err := r.Get(ctx, req.NamespacedName, instance)
	// Error reading the object - requeue the request.
	if err != nil {
		// handle deletion of resource
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("resource has been deleted", "req", req.Name, "got", instance.Name)
			return ctrl.Result{Requeue: false}, nil
		}

		r.Log.Error(err, "requeueing event since there was an error reading object")
		return ctrl.Result{Requeue: true}, err
	}

	// If the resources are to be deleted, first check to see if the
	// deletion timestamp pointer is not nil. A non-nil value indicates
	// someone or something has triggered the deletion.
	if instance.DeletionTimestamp != nil {
		return r.finalizeNFDOperator(ctx, instance, finalizer)
	}

	// If the finalizer doesn't exist, add it.
	if !r.hasFinalizer(instance, finalizer) {
		return r.addFinalizer(ctx, instance, finalizer)
	}

	// apply components
	r.Log.Info("Ready to apply components")

	nfd.init(r, instance)

	for {
		err := nfd.step()
		if err != nil {
			return reconcile.Result{}, err
		}
		if nfd.last() {
			break
		}
	}

	return ctrl.Result{}, nil
}
