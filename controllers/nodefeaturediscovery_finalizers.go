/*
Copyright 2022 The Kubernetes Authors.

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
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	nfdv1 "sigs.k8s.io/node-feature-discovery-operator/api/v1"
)

var (
	RetryInterval = time.Second * 5
	Timeout       = time.Second * 30
)

// finalizeNFDOperand finalizes an NFD Operand instance
func (r *NodeFeatureDiscoveryReconciler) finalizeNFDOperand(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery, finalizer string) (ctrl.Result, error) {
	klog.Info("Attempting to delete NFD operator components")
	if err := r.deleteComponents(ctx, instance); err != nil {
		klog.Error(err, "Failed to delete one or more components")
		return ctrl.Result{}, err
	}

	// Check if all components are deleted. If they're not,
	// then call the reconciler but wait 10 seconds before
	// checking again.
	klog.Info("Deletion appears to have succeeded, but running a secondary check to ensure resources are cleaned up")
	if r.doComponentsExist(ctx, instance) {
		klog.Info("Some components still exist. Requeueing deletion request.")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if instance.Spec.PruneOnDelete {
		klog.Info("Deleting NFD labels and NodeFeature CRs from cluster")
		if err := deployPrune(ctx, r, instance); err != nil {
			klog.Error(err, "Failed to delete NFD labels and NodeFeature CRs from cluster")
			return ctrl.Result{}, err
		}
	} else {
		klog.Warning("PruneOnDelete is disabled, NFD labels and NodeFeature CRs will not be deleted from cluster")
	}

	// If all components are deleted, then remove the finalizer
	klog.Info("Secondary check passed. Removing finalizer if it exists.")
	if r.hasFinalizer(instance, finalizer) {
		r.removeFinalizer(instance, finalizer)
		if err := r.Update(ctx, instance); err != nil {
			if k8serrors.IsNotFound(err) {
				return ctrl.Result{Requeue: false}, nil
			}
			klog.Info("Finalizer was found, but removing it was unsuccessful. Requeueing deletion request.")
			return ctrl.Result{}, nil
		}

		klog.Info("Finalizer was found and successfully removed.")
		return ctrl.Result{Requeue: false}, nil
	}

	klog.Info("Finalizer does not exist, but resource deletion successful.")
	return ctrl.Result{Requeue: false}, nil
}

// addFinalizer adds a finalizer for the NFD operand.
func (r *NodeFeatureDiscoveryReconciler) addFinalizer(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery, finalizer string) (ctrl.Result, error) {
	instance.Finalizers = append(instance.Finalizers, finalizer)
	instance.Status.Conditions = r.getProgressingConditions("DeploymentStarting", "Deployment is starting")
	if err := r.Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// we exit reconcile loop because we will have additional update reconcile
	return ctrl.Result{Requeue: false}, nil
}

// hasFinalizer determines if the operand has a certain finalizer.
func (r *NodeFeatureDiscoveryReconciler) hasFinalizer(instance *nfdv1.NodeFeatureDiscovery, finalizer string) bool {
	for _, f := range instance.Finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

// removeFinalizer removes a finalizer from the operand.
func (r *NodeFeatureDiscoveryReconciler) removeFinalizer(instance *nfdv1.NodeFeatureDiscovery, finalizer string) {
	var finalizers []string

	for _, f := range instance.Finalizers {
		if f != finalizer {
			finalizers = append(finalizers, f)
		}
	}

	instance.Finalizers = finalizers
}

// deleteComponents deletes all of the NFD operand components.
func (r *NodeFeatureDiscoveryReconciler) deleteComponents(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) error {
	// Update CRD status to notify instance is undergoing deletion
	_, _ = r.updateProgressingCondition(instance, "finalizers", "Foreground-Deletion")

	// If NFD-Topology-Updater was requested
	if instance.Spec.TopologyUpdater {
		// Attempt to delete Topology DaemonSet
		err := wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
			err = r.deleteDaemonSet(ctx, instance.ObjectMeta.Namespace, nfdTopologyUpdaterApp)
			if err != nil {
				return false, interpretError(err, "Topology DaemonSet")
			}
			klog.Info("nfd-topology-updater DaemonSet resource has been deleted.")
			return true, nil
		})
		if err != nil {
			return err
		}
		// Attempt to delete the ClusterRole
		err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
			err = r.deleteClusterRole(ctx, instance.ObjectMeta.Namespace, nfdTopologyUpdaterApp)
			if err != nil {
				return false, interpretError(err, "nfd-Topology-Updater ClusterRole")
			}
			klog.Info("nfd-Topology-Updater ClusterRole resource has been deleted.")
			return true, nil
		})
		if err != nil {
			return err
		}
		// Attempt to delete the ClusterRoleBinding
		err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
			err = r.deleteClusterRoleBinding(ctx, instance.ObjectMeta.Namespace, nfdTopologyUpdaterApp)
			if err != nil {
				return false, interpretError(err, "ClusterRoleBinding")
			}
			klog.Info("ClusterRoleBinding ", nfdTopologyUpdaterApp, " resource has been deleted.")
			return true, nil
		})
		if err != nil {
			return err
		}
		// Attempt to delete the Worker ServiceAccount
		err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
			err = r.deleteServiceAccount(ctx, instance.ObjectMeta.Namespace, nfdTopologyUpdaterApp)
			if err != nil {
				return false, interpretError(err, "worker ServiceAccount")
			}
			klog.Info("nfd-Topology-Updater ServiceAccount resource has been deleted.")
			return true, nil
		})
		if err != nil {
			return err
		}
	}

	// Attempt to delete worker DaemonSet
	err := wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteDaemonSet(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp)
		if err != nil {
			return false, interpretError(err, "worker DaemonSet")
		}
		klog.Info("Worker DaemonSet resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete master Deployment
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteDeployment(ctx, instance.ObjectMeta.Namespace, nfdMasterApp)
		if err != nil {
			return false, interpretError(err, "master Deployment")
		}
		klog.Info("Master Deployment resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the Service
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteService(ctx, instance.ObjectMeta.Namespace, nfdMasterApp)
		if err != nil {
			return false, interpretError(err, "nfd-master Service")
		}
		klog.Info("nfd-master Service resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the Role
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteRole(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp)
		if err != nil {
			return false, interpretError(err, "nfd-worker Role")
		}
		klog.Info("nfd-worker Role resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the ClusterRole
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteClusterRole(ctx, instance.ObjectMeta.Namespace, nfdMasterApp)
		if err != nil {
			return false, interpretError(err, "nfd-master ClusterRole")
		}
		klog.Info("nfd-master ClusterRole resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the RoleBinding
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteRoleBinding(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp)
		if err != nil {
			return false, interpretError(err, "nfd-worker RoleBinding")
		}
		klog.Info("nfd-worker RoleBinding resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the ClusterRoleBinding
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteClusterRoleBinding(ctx, instance.ObjectMeta.Namespace, nfdMasterApp)
		if err != nil {
			return false, interpretError(err, "ClusterRoleBinding")
		}
		klog.Info("ClusterRoleBinding ", nfdMasterApp, " resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the Worker ServiceAccount
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteServiceAccount(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp)
		if err != nil {
			return false, interpretError(err, "worker ServiceAccount")
		}
		klog.Info("Worker ServiceAccount resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the Master ServiceAccount
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteServiceAccount(ctx, instance.ObjectMeta.Namespace, nfdMasterApp)
		if err != nil {
			return false, interpretError(err, "master ServiceAccount")
		}
		klog.Info("Master ServiceAccount resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the Worker config map
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteConfigMap(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp)
		if err != nil {
			return false, interpretError(err, "nfd-worker config map")
		}
		klog.Info("nfd-worker config map resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// doComponentsExist checks to see if any of the operand components exist.
func (r *NodeFeatureDiscoveryReconciler) doComponentsExist(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) bool {
	// Attempt to find the worker DaemonSet
	if _, err := r.getDaemonSet(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to find the master Deployment
	if _, err := r.getDeployment(ctx, instance.ObjectMeta.Namespace, nfdMasterApp); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the Service
	if _, err := r.getService(ctx, instance.ObjectMeta.Namespace, nfdMasterApp); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the Role
	if _, err := r.getRole(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the ClusterRole
	if _, err := r.getClusterRole(ctx, instance.ObjectMeta.Namespace, nfdMasterApp); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the RoleBinding
	if _, err := r.getRoleBinding(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the ClusterRoleBinding
	if _, err := r.getClusterRoleBinding(ctx, instance.ObjectMeta.Namespace, nfdMasterApp); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the Worker ServiceAccount
	if _, err := r.getServiceAccount(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the Master ServiceAccount
	if _, err := r.getServiceAccount(ctx, instance.ObjectMeta.Namespace, nfdMasterApp); !k8serrors.IsNotFound(err) {
		return true
	}

	if instance.Spec.TopologyUpdater {
		// Attempt to find the topology-updater DaemonSet
		if _, err := r.getDaemonSet(ctx, instance.ObjectMeta.Namespace, nfdTopologyUpdaterApp); !k8serrors.IsNotFound(err) {
			return true
		}
		// Attempt to get the Worker ServiceAccount
		if _, err := r.getServiceAccount(ctx, instance.ObjectMeta.Namespace, nfdTopologyUpdaterApp); !k8serrors.IsNotFound(err) {
			return true
		}
		// Attempt to get the ClusterRole
		if _, err := r.getClusterRole(ctx, instance.ObjectMeta.Namespace, nfdTopologyUpdaterApp); !k8serrors.IsNotFound(err) {
			return true
		}
		// Attempt to get the ClusterRoleBinding
		if _, err := r.getClusterRoleBinding(ctx, instance.ObjectMeta.Namespace, nfdTopologyUpdaterApp); !k8serrors.IsNotFound(err) {
			return true
		}
	}

	return false
}

// deployPrune deploys nfd-master with --prune option
// to remove labels and NodeFeature CRs
func deployPrune(ctx context.Context, r *NodeFeatureDiscoveryReconciler, instance *nfdv1.NodeFeatureDiscovery) error {
	res, ctrl := addResourcesControls("/opt/nfd/prune")
	n := NFD{
		rec: r,
		ins: instance,
		idx: 0,
	}

	n.controls = append(n.controls, ctrl)
	n.resources = append(n.resources, res)

	// Run through all control functions, return an error on any NotReady resource.
	for {
		err := n.step()
		if err != nil {
			return err
		}
		if n.last() {
			break
		}
	}

	// wait until job is finished and then delete it
	err := wait.Poll(RetryInterval, time.Minute*3, func() (done bool, err error) {
		job, err := r.getJob(ctx, instance.ObjectMeta.Namespace, nfdPruneApp)
		if err != nil {
			return false, err
		}
		if job.Status.Succeeded > 0 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	// delete job and RBAC objects
	// Attempt to delete the Job
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteJob(ctx, instance.ObjectMeta.Namespace, nfdPruneApp)
		if err != nil {
			return false, interpretError(err, "Prune Job")
		}
		klog.Info("nfd-prune Job resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}
	// Attempt to delete the ServiceAccount
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteServiceAccount(ctx, instance.ObjectMeta.Namespace, nfdPruneApp)
		if err != nil {
			return false, interpretError(err, "Prune ServiceAccount")
		}
		klog.Info("nfd-prune ServiceAccount resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the ClusterRole
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteClusterRole(ctx, instance.ObjectMeta.Namespace, nfdPruneApp)
		if err != nil {
			return false, interpretError(err, "Prune ClusterRole")
		}
		klog.Info("nfd-prune ClusterRole resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the ClusterRoleBinding
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteClusterRoleBinding(ctx, instance.ObjectMeta.Namespace, nfdPruneApp)
		if err != nil {
			return false, interpretError(err, "Prune ClusterRoleBinding")
		}
		klog.Info("nfd-prune ClusterRoleBinding resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// interpretError determines if a resource has already been
// (successfully) deleted
func interpretError(err error, resourceName string) error {
	if k8serrors.IsNotFound(err) {
		klog.Info("Resource ", resourceName, " has been deleted.")
		return nil
	}
	return err
}
