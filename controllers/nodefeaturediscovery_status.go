/*
Copyright 2021 The Kubernetes Authors.

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
	"errors"
	"time"

	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nfdv1 "sigs.k8s.io/node-feature-discovery-operator/api/v1"
)

const (
	nfdWorkerApp          string = "nfd-worker"
	nfdMasterApp          string = "nfd-master"
	nfdTopologyUpdaterApp string = "nfd-topology-updater"
)

const (
	// Resource is missing
	conditionFailedGettingNFDWorkerConfig                  = "FailedGettingNFDWorkerConfig"
	conditionFailedGettingNFDWorkerServiceAccount          = "FailedGettingNFDWorkerServiceAccount"
	conditionFailedGettingNFDTopologyUpdaterServiceAccount = "FailedGettingNFDTopoloGyUpdaterServiceAccount"
	conditionFailedGettingNFDMasterServiceAccount          = "FailedGettingNFDMasterServiceAccount"
	conditionFailedGettingNFDService                       = "FailedGettingNFDService"
	conditionFailedGettingNFDWorkerDaemonSet               = "FailedGettingNFDWorkerDaemonSet"
	conditionFailedGettingNFDMasterDaemonSet               = "FailedGettingNFDMasterDaemonSet"
	conditionFailedGettingNFDRoleBinding                   = "FailedGettingNFDRoleBinding"
	conditionFailedGettingNFDClusterRoleBinding            = "FailedGettingNFDClusterRole"

	// Resource degraded
	conditionNFDWorkerConfigDegraded                  = "NFDWorkerConfigResourceDegraded"
	conditionNFDWorkerServiceAccountDegraded          = "NFDWorkerServiceAccountDegraded"
	conditionNFDTopologyUpdaterServiceAccountDegraded = "NFDTopologyUpdaterServiceAccountDegraded"
	conditionNFDMasterServiceAccountDegraded          = "NFDMasterServiceAccountDegraded"
	conditionNFDServiceDegraded                       = "NFDServiceDegraded"
	conditionNFDWorkerDaemonSetDegraded               = "NFDWorkerDaemonSetDegraded"
	conditionNFDTopologyUpdaterDaemonSetDegraded      = "NFDTopologyUpdaterDaemonSetDegraded"
	conditionNFDMasterDaemonSetDegraded               = "NFDMasterDaemonSetDegraded"
	conditionNFDRoleDegraded                          = "NFDRoleDegraded"
	conditionNFDRoleBindingDegraded                   = "NFDRoleBindingDegraded"
	conditionNFDClusterRoleDegraded                   = "NFDClusterRoleDegraded"
	conditionNFDClusterRoleBindingDegraded            = "NFDClusterRoleBindingDegraded"

	// Unknown errors. (Catch all)
	errorNFDWorkerDaemonSetUnknown = "NFDWorkerDaemonSetCorrupted"
	errorNFDMasterDaemonSetUnknown = "NFDMasterDaemonSetCorrupted"

	// More nodes are listed as "ready" than selected
	errorTooManyNFDWorkerDaemonSetReadyNodes = "NFDWorkerDaemonSetHasMoreNodesThanScheduled"
	errorTooManyNFDMasterDaemonSetReadyNodes = "NFDMasterDaemonSetHasMoreNodesThanScheduled"

	// DaemonSet warnings (for "Progressing" conditions)
	warningNumberOfReadyNodesIsLessThanScheduled = "warningNumberOfReadyNodesIsLessThanScheduled"
	warningNFDWorkerDaemonSetProgressing         = "warningNFDWorkerDaemonSetProgressing"
	warningNFDMasterDaemonSetProgressing         = "warningNFDMasterDaemonSetProgressing"

	// ConditionAvailable indicates that the resources maintained by the operator,
	// is functional and available in the cluster.
	ConditionAvailable string = "Available"

	// ConditionProgressing indicates that the operator is actively making changes to the resources maintained by the
	// operator
	ConditionProgressing string = "Progressing"

	// ConditionDegraded indicates that the resources maintained by the operator are not functioning completely.
	// An example of a degraded state would be if not all pods in a deployment were running.
	// It may still be available, but it is degraded
	ConditionDegraded string = "Degraded"

	// ConditionUpgradeable indicates whether the resources maintained by the operator are in a state that is safe to upgrade.
	// When `False`, the resources maintained by the operator should not be upgraded and the
	// message field should contain a human readable description of what the administrator should do to
	// allow the operator to successfully update the resources maintained by the operator.
	ConditionUpgradeable string = "Upgradeable"
)

// updateStatus is used to update the status of a resource (e.g., degraded,
// available, etc.)
func (r *NodeFeatureDiscoveryReconciler) updateStatus(nfd *nfdv1.NodeFeatureDiscovery, condition []metav1.Condition) error {
	// The actual 'nfd' object should *not* be modified when trying to
	// check the object's status. This variable is a dummy variable used
	// to set temporary conditions.
	nfdCopy := nfd.DeepCopy()

	// If a set of conditions exists, then it should be added to the
	// 'nfd' Copy.
	if condition != nil {
		nfdCopy.Status.Conditions = condition
	}

	// Next step is to check if we need to update the status
	modified := false

	// Because there are only four possible conditions (degraded, available,
	// updatable, and progressing), it isn't necessary to check if old
	// conditions should be removed.
	for _, newCondition := range nfdCopy.Status.Conditions {
		oldCondition := meta.FindStatusCondition(nfd.Status.Conditions, newCondition.Type)
		if oldCondition == nil {
			modified = true
			break
		}
		// Ignore timestamps to avoid infinite reconcile loops
		if oldCondition.Status != newCondition.Status ||
			oldCondition.Reason != newCondition.Reason ||
			oldCondition.Message != newCondition.Message {
			modified = true
			break
		}
	}

	// If nothing has been modified, then return nothing. Even if the list
	// of 'conditions' is not empty, it should not be counted as an update
	// if it was already counted as an update before.
	if !modified {
		return nil
	}
	return r.Status().Update(context.TODO(), nfdCopy)
}

// updateDegradedCondition is used to mark a given resource as "degraded" so that
// the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateDegradedCondition(nfd *nfdv1.NodeFeatureDiscovery, reason, message string) (ctrl.Result, error) {
	degradedCondition := r.getDegradedConditions(reason, message)
	if err := r.updateStatus(nfd, degradedCondition); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

// updateProgressingCondition is used to mark a given resource as "progressing" so
// that the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateProgressingCondition(nfd *nfdv1.NodeFeatureDiscovery, reason, message string) (ctrl.Result, error) {
	progressingCondition := r.getProgressingConditions(reason, message)
	if err := r.updateStatus(nfd, progressingCondition); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, nil
}

// getAvailableConditions returns a list of Condition objects and marks
// every condition as FALSE except for ConditionAvailable so that the
// reconciler can determine that the resource is available.
func (r *NodeFeatureDiscoveryReconciler) getAvailableConditions() []metav1.Condition {
	now := time.Now()
	return []metav1.Condition{
		{
			Type:               ConditionAvailable,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               ConditionUpgradeable,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               ConditionProgressing,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               ConditionDegraded,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
	}
}

// getDegradedConditions returns a list of conditions.Condition objects and marks
// every condition as FALSE except for conditions.ConditionDegraded so that the
// reconciler can determine that the resource is degraded.
func (r *NodeFeatureDiscoveryReconciler) getDegradedConditions(reason string, message string) []metav1.Condition {
	now := time.Now()
	return []metav1.Condition{
		{
			Type:               ConditionAvailable,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               ConditionUpgradeable,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               ConditionProgressing,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               ConditionDegraded,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
	}
}

// getProgressingConditions returns a list of Condition objects and marks
// every condition as FALSE except for ConditionProgressing so that the
// reconciler can determine that the resource is progressing.
func (r *NodeFeatureDiscoveryReconciler) getProgressingConditions(reason string, message string) []metav1.Condition {
	now := time.Now()
	return []metav1.Condition{
		{
			Type:               ConditionAvailable,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               ConditionUpgradeable,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               ConditionProgressing,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
		{
			Type:               ConditionDegraded,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
	}
}

// The status of the resource (available, upgradeable, progressing, or
// degraded).
type Status struct {
	// Is the resource available, upgradable, etc.?
	isAvailable   bool
	isProgressing bool
	isDegraded    bool
}

// initializeDegradedStatus initializes the status struct to degraded
func initializeDegradedStatus() Status {
	return Status{
		isAvailable:   false,
		isProgressing: false,
		isDegraded:    true,
	}
}

// getWorkerDaemonSetConditions is a wrapper around "getDaemonSetConditions" for
// worker DaemonSets
func (r *NodeFeatureDiscoveryReconciler) getWorkerDaemonSetConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getDaemonSetConditions(ctx, instance, nfdWorkerApp)
}

// getTopologyUpdaterDaemonSetConditions is a wrapper around "getDaemonSetConditions" for
// worker DaemonSets
func (r *NodeFeatureDiscoveryReconciler) getTopologyUpdaterDaemonSetConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getDaemonSetConditions(ctx, instance, nfdTopologyUpdaterApp)
}

// getMasterDaemonSetConditions is a wrapper around "getDaemonSetConditions" for
// master DaemonSets
func (r *NodeFeatureDiscoveryReconciler) getMasterDaemonSetConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getDaemonSetConditions(ctx, instance, nfdMasterApp)
}

// getDaemonSetConditions gets the current status of a DaemonSet. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getDaemonSetConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery, nfdAppName string) (Status, error) {
	// Initialize the resource's status to 'Degraded'
	status := initializeDegradedStatus()

	ds, err := r.getDaemonSet(ctx, instance.ObjectMeta.Namespace, nfdAppName)
	if err != nil {
		return status, err
	}

	// Index the DaemonSet status. (Note: there is no "Conditions" array here.)
	dsStatus := ds.Status

	// Index the relevant values from here
	numberReady := dsStatus.NumberReady
	currentNumberScheduled := dsStatus.CurrentNumberScheduled
	numberDesired := dsStatus.DesiredNumberScheduled
	numberUnavailable := dsStatus.NumberUnavailable

	// If the number desired is zero or the number of unavailable nodes is zero,
	// then we have a problem because we should at least see 1 pod per node
	if numberDesired == 0 {
		if nfdAppName == nfdWorkerApp {
			return status, errors.New(errorNFDWorkerDaemonSetUnknown)
		}
		return status, errors.New(errorNFDMasterDaemonSetUnknown)
	}
	if numberUnavailable > 0 {
		status.isProgressing = true
		status.isDegraded = false
		if nfdAppName == nfdWorkerApp {
			return status, errors.New(warningNFDWorkerDaemonSetProgressing)
		}
		return status, errors.New(warningNFDMasterDaemonSetProgressing)
	}

	// If there are none scheduled, then we have a problem because we should
	// at least see 1 pod per node, even after the scheduling happens.
	if currentNumberScheduled == 0 {
		if nfdAppName == nfdWorkerApp {
			return status, errors.New(conditionNFDWorkerDaemonSetDegraded)
		}
		return status, errors.New(conditionNFDMasterDaemonSetDegraded)
	}

	// Just check in case the number of "ready" nodes is greater than the
	// number of scheduled ones (for whatever reason)
	if numberReady > currentNumberScheduled {
		status.isDegraded = false
		if nfdAppName == nfdWorkerApp {
			return status, errors.New(errorTooManyNFDWorkerDaemonSetReadyNodes)
		}
		return status, errors.New(errorTooManyNFDMasterDaemonSetReadyNodes)
	}

	// If we have less than the number of scheduled pods, then the DaemonSet
	// is in progress
	if numberReady < currentNumberScheduled {
		status.isProgressing = true
		status.isDegraded = false
		return status, errors.New(warningNumberOfReadyNodesIsLessThanScheduled)
	}

	// If all nodes are ready, then update the status to be "isAvailable"
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}

// getServiceConditions gets the current status of a Service. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getServiceConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing Service from the reconciler
	_, err := r.getService(ctx, instance.ObjectMeta.Namespace, nfdMasterApp)

	// If the Service could not be obtained, then it is degraded
	if err != nil {
		return status, errors.New(conditionNFDServiceDegraded)
	}

	// If we could get the Service, then it is not empty and it exists
	status.isAvailable = true
	status.isDegraded = false

	return status, nil

}

// getWorkerConfigConditions gets the current status of a worker config. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getWorkerConfigConditions(n NFD) (Status, error) {
	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing ConfigMap from the reconciler
	wc := n.ins.Spec.WorkerConfig.ConfigData

	// If 'wc' is nil, then the resource hasn't been (re)created yet
	if wc == "" {
		return status, errors.New(conditionNFDWorkerConfigDegraded)
	}

	// If we could get the WorkerConfig, then it is not empty and it exists
	status.isDegraded = false
	status.isAvailable = true

	return status, nil
}

// getRoleConditions gets the current status of a Role. If an error occurs, this
// function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getRoleConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing Role from the reconciler
	_, err := r.getRole(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp)

	// If the error is not nil, then the Role hasn't been (re)created yet
	if err != nil {
		return status, errors.New(conditionNFDRoleDegraded)
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}

// getRoleBindingConditions gets the current status of a RoleBinding. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getRoleBindingConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing RoleBinding from the reconciler
	_, err := r.getRoleBinding(ctx, instance.ObjectMeta.Namespace, nfdWorkerApp)

	// If the error is not nil, then the RoleBinding hasn't been (re)created yet
	if err != nil {
		return status, errors.New(conditionNFDRoleBindingDegraded)
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}

// getMasterClusterRoleConditions is a wrapper around "getClusterRoleConditions" for
// worker service account.
func (r *NodeFeatureDiscoveryReconciler) getMasterClusterRoleConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getClusterRoleConditions(ctx, instance, nfdMasterApp)
}

// getTopologyUpdaterClusterRoleConditions is a wrapper around "getClusterRoleConditions" for
// worker service account.
func (r *NodeFeatureDiscoveryReconciler) getTopologyUpdaterClusterRoleConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getClusterRoleConditions(ctx, instance, nfdTopologyUpdaterApp)
}

// geClusterRoleConditions gets the current status of a ClusterRole. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getClusterRoleConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery, nfdAppName string) (Status, error) {
	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing ClusterRole from the reconciler
	_, err := r.getClusterRole(ctx, instance.ObjectMeta.Namespace, nfdAppName)

	// If 'clusterRole' is nil, then it hasn't been (re)created yet
	if err != nil {
		return status, errors.New(conditionNFDClusterRoleDegraded)
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}

// getMasterClusterRoleBindingConditions is a wrapper around "getServiceAccountConditions" for
// worker service account.
func (r *NodeFeatureDiscoveryReconciler) getMasterClusterRoleBindingConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getServiceAccountConditions(ctx, instance, nfdMasterApp)
}

// getTopologyUpdaterClusterRoleBindingConditions is a wrapper around "getServiceAccountConditions" for
// worker service account.
func (r *NodeFeatureDiscoveryReconciler) getTopologyUpdaterClusterRoleBindingConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getClusterRoleBindingConditions(ctx, instance, nfdTopologyUpdaterApp)
}

// getClusterRoleBindingConditions gets the current status of a ClusterRoleBinding.
// If an error occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBindingConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery, nfdAppName string) (Status, error) {
	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing ClusterRoleBinding from the reconciler
	_, err := r.getClusterRoleBinding(ctx, instance.ObjectMeta.Namespace, nfdAppName)

	// If the error is not nil, then the ClusterRoleBinding hasn't been (re)created
	// yet
	if err != nil {
		return status, errors.New(conditionNFDClusterRoleBindingDegraded)
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}

// getWorkerServiceAccountConditions is a wrapper around "getServiceAccountConditions" for
// worker service account.
func (r *NodeFeatureDiscoveryReconciler) getWorkerServiceAccountConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getServiceAccountConditions(ctx, instance, nfdWorkerApp)
}

// getTopologyUpdaterServiceAccountConditions is a wrapper around "getServiceAccountConditions" for
// worker service account.
func (r *NodeFeatureDiscoveryReconciler) getTopologyUpdaterServiceAccountConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getServiceAccountConditions(ctx, instance, nfdTopologyUpdaterApp)
}

// getMasterServiceAccountConditions is a wrapper around "getServiceAccountConditions" for
// master service account.
func (r *NodeFeatureDiscoveryReconciler) getMasterServiceAccountConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery) (Status, error) {
	return r.getServiceAccountConditions(ctx, instance, nfdMasterApp)
}

// getServiceAccountConditions gets the current status of a ServiceAccount. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getServiceAccountConditions(ctx context.Context, instance *nfdv1.NodeFeatureDiscovery, nfdAppName string) (Status, error) {
	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the service account from the reconciler
	_, err := r.getServiceAccount(ctx, instance.ObjectMeta.Namespace, nfdAppName)

	// If the error is not nil, then the ServiceAccount hasn't been (re)created yet
	if err != nil {
		switch nfdAppName {
		case nfdWorkerApp:
			return status, errors.New(conditionNFDWorkerServiceAccountDegraded)
		case nfdMasterApp:
			return status, errors.New(conditionNFDMasterServiceAccountDegraded)
		case nfdTopologyUpdaterApp:
			return status, errors.New(conditionNFDTopologyUpdaterServiceAccountDegraded)
		}
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}
