/*
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
	ctrl "sigs.k8s.io/controller-runtime"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
)

//"sigs.k8s.io/controller-runtime/pkg/reconcile"

var (
	RetryInterval = time.Second * 5
	Timeout       = time.Second * 30
)

// finalizeNFDOperator finalizes an NFD Operator instance
func (r *NodeFeatureDiscoveryReconciler) finalizeNFDOperator(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery, finalizer string) (ctrl.Result, error) {

	// Attempt to delete all components. If it fails, return
	// a warning letting users know the deletion failed, and
	// then call the reconciler once more to see if the error
	// can be corrected.
	log.Info("Attempting to delete NFD operator components")
	if err := r.deleteComponents(ctx); err != nil {
		r.Log.Error(err, "Failed to delete one or more components")
		return ctrl.Result{}, err
	}

	// Check if all components are deleted. If they're not,
	// then call the reconciler but wait 10 seconds before
	// checking again.
	log.Info("Deletion appears to have succeeded, but running a secondary check to ensure resources are cleaned up")
	if r.doComponentsExist(ctx) == true {
		log.Info("Some components still exist. Requeueing deletion request.")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// If all components are deleted, then remove the finalizer
	log.Info("Secondary check passed. Removing finalizer if it exists.")
	if r.hasFinalizer(instance, finalizer) == true {
		r.removeFinalizer(instance, finalizer)
		if err := r.Update(ctx, instance); err != nil {
			if k8serrors.IsNotFound(err) {
				return ctrl.Result{Requeue: false}, nil
			}
			log.Info("Finalizer was found, but removing it was unsuccessful. Requeueing deletion request.")
			return ctrl.Result{}, nil
		}

		log.Info("Finalizer was found and successfully removed.")
		return ctrl.Result{Requeue: false}, nil
	}

	log.Info("Finalizer does not exist, but resource deletion succesful.")
	return ctrl.Result{Requeue: false}, nil
}

// addFinalizer adds a finalizer to the NFD Operator instance
func (r *NodeFeatureDiscoveryReconciler) addFinalizer(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery, finalizer string) (ctrl.Result, error) {

	// Add the defined finalizer as a finalizer to the instance if it does not exist
	instance.Finalizers = append(instance.Finalizers, finalizer)
	instance.Status.Conditions = r.getProgressingConditions("DeploymentStarting", "Deployment is starting")
	if err := r.Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// we exit reconcile loop because we will have additional update reconcile
	return ctrl.Result{Requeue: false}, nil
}

// hasFinalizer determines if the operator instance has a specific
// finalizer value, which is defined by the parameter 'finalizer'
func (r *NodeFeatureDiscoveryReconciler) hasFinalizer(instance *nfdv1.NodeFeatureDiscovery, finalizer string) bool {

	if len(instance.Finalizers) == 0 {
		return false
	}

	// The instance will have a list of finalizers under its
	// `metav1.ObjectMeta` reference
	for _, f := range instance.Finalizers {

		// If the current finalizer in the list matches the
		// 'finalizer' parameter, then the operator does have
		// the desired finalizer, so return "true"
		if f == finalizer {
			return true
		}
	}

	// Return false, as the finalizer was not found in the list.
	return false
}

// removeFinalizer removes a finalizer from the operator's instance
func (r *NodeFeatureDiscoveryReconciler) removeFinalizer(instance *nfdv1.NodeFeatureDiscovery, finalizer string) {

	// 'finalizers' will contain a list of all the finalizers for
	// the NFD operator instance, except for the finalizer that
	// is being removed. (The finalizer to remove is defined with
	// this function's parameter 'finalizer'.)
	var finalizers []string

	// The instance will have a list of finalizers under its
	// `metav1.ObjectMeta` reference
	for _, f := range instance.Finalizers {

		// If the current finalizer in the list matches the
		// 'finalizer' parameter, then we want to remove it.
		// However, rather than delete from the list, it is
		// more efficient to just create a new list and set
		// the 'Finalizers' attribute to that new list. Thus,
		// this part of the loop skips the addition of the
		// finalizer we want to remove.
		if f == finalizer {
			continue
		}
		finalizers = append(finalizers, f)
	}

	// Update the 'Finalizers' attribute to point to the newly
	// updated list.
	instance.Finalizers = finalizers
}

// deleteComponents deletes all of the NFD operator's components
func (r *NodeFeatureDiscoveryReconciler) deleteComponents(ctx context.Context) error {

	// Attempt to delete worker DaemonSet
	err := wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteDaemonSet(ctx, nfdNamespace, workerName)
		if err != nil {
			return false, interpretError(err, "worker DaemonSet")
		}
		log.Info("Worker DaemonSet resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete master DaemonSet
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteDaemonSet(ctx, nfdNamespace, masterName)
		if err != nil {
			return false, interpretError(err, "master DaemonSet")
		}
		log.Info("Master DaemonSet resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the Service
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteService(ctx, nfdNamespace, masterName)
		if err != nil {
			return false, interpretError(err, "Service")
		}
		log.Info("Service resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the Role
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteRole(ctx, nfdNamespace, workerName)
		if err != nil {
			return false, interpretError(err, "Role")
		}
		log.Info("Role resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the ClusterRole
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteClusterRole(ctx, nfdNamespace, masterName)
		if err != nil {
			return false, interpretError(err, "ClusterRole")
		}
		log.Info("ClusterRole resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the RoleBinding
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteRoleBinding(ctx, nfdNamespace, workerName)
		if err != nil {
			return false, interpretError(err, "RoleBinding")
		}
		log.Info("RoleBinding resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the ClusterRoleBinding
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteClusterRoleBinding(ctx, nfdNamespace, workerName)
		if err != nil {
			return false, interpretError(err, "ClusterRoleBinding")
		}
		log.Info("ClusterRoleBinding resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the Worker ServiceAccount
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteServiceAccount(ctx, nfdNamespace, workerName)
		if err != nil {
			return false, interpretError(err, "worker ServiceAccount")
		}
		log.Info("Worker ServiceAccount resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the Master ServiceAccount
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteServiceAccount(ctx, nfdNamespace, masterName)
		if err != nil {
			return false, interpretError(err, "master ServiceAccount")
		}
		log.Info("Master ServiceAccount resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	// Attempt to delete the SecurityContextConstraints
	err = wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = r.deleteSecurityContextConstraints(ctx, nfdNamespace, workerName)
		if err != nil {
			return false, interpretError(err, "SecurityContextConstraints")
		}
		log.Info("SecurityContextConstraints resource has been deleted.")
		return true, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// doComponentsExist checks to see if any of the NFD Operator's
// components exist. If they do, then return 'true' to let the
// user know that all components have NOT been deleted successfully
func (r *NodeFeatureDiscoveryReconciler) doComponentsExist(ctx context.Context) bool {

	// Attempt to find the worker DaemonSet
	if _, err := r.getDaemonSet(ctx, nfdNamespace, workerName); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to find the master DaemonSet
	if _, err := r.getDaemonSet(ctx, nfdNamespace, masterName); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the Service
	if _, err := r.getService(ctx, nfdNamespace, masterName); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the Role
	if _, err := r.getRole(ctx, nfdNamespace, workerName); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the ClusterRole
	if _, err := r.getClusterRole(ctx, nfdNamespace, masterName); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the RoleBinding
	if _, err := r.getRoleBinding(ctx, nfdNamespace, workerName); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the ClusterRoleBinding
	if _, err := r.getClusterRoleBinding(ctx, nfdNamespace, masterName); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the Worker ServiceAccount
	if _, err := r.getServiceAccount(ctx, nfdNamespace, workerName); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the Master ServiceAccount
	if _, err := r.getServiceAccount(ctx, nfdNamespace, masterName); !k8serrors.IsNotFound(err) {
		return true
	}

	// Attempt to get the SecurityContextConstraints
	if _, err := r.getSecurityContextConstraints(ctx, nfdNamespace, workerName); !k8serrors.IsNotFound(err) {
		return true
	}

	return false
}

// interpretError determines if a resource has already been
// (successfully) deleted
func interpretError(err error, resourceName string) error {

	if k8serrors.IsNotFound(err) {
		log.Info("Resource ", resourceName, " has been deleted.")
		return nil
	}
	return err
}
