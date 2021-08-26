package controllers

import (
	"context"
	"errors"
	"time"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nfdv1 "github.com/kubernetes-sigs/node-feature-discovery-operator/api/v1"
)

// nodeType is either 'worker' or 'master'
type nodeType int

const (
	worker       nodeType = 0
	master       nodeType = 1
	nfdNamespace          = "node-feature-discovery-operator"
	workerName            = "nfd-worker"
	masterName            = "nfd-master"
)

const (
	// Resource is missing
	conditionFailedGettingNFDWorkerConfig         = "FailedGettingNFDWorkerConfig"
	conditionFailedGettingNFDWorkerServiceAccount = "FailedGettingNFDWorkerServiceAccount"
	conditionFailedGettingNFDMasterServiceAccount = "FailedGettingNFDMasterServiceAccount"
	conditionFailedGettingNFDService              = "FailedGettingNFDService"
	conditionFailedGettingNFDWorkerDaemonSet      = "FailedGettingNFDWorkerDaemonSet"
	conditionFailedGettingNFDMasterDaemonSet      = "FailedGettingNFDMasterDaemonSet"
	conditionFailedGettingNFDRoleBinding          = "FailedGettingNFDRoleBinding"
	conditionFailedGettingNFDScc                  = "FailedGettingNFDSecurityContextConstraints"

	// Resource degraded
	conditionNFDWorkerConfigDegraded               = "NFDWorkerConfigResourceDegraded"
	conditionNFDWorkerServiceAccountDegraded       = "NFDWorkerServiceAccountDegraded"
	conditionNFDMasterServiceAccountDegraded       = "NFDMasterServiceAccountDegraded"
	conditionNFDServiceDegraded                    = "NFDServiceDegraded"
	conditionNFDWorkerDaemonSetDegraded            = "NFDWorkerDaemonSetDegraded"
	conditionNFDMasterDaemonSetDegraded            = "NFDMasterDaemonSetDegraded"
	conditionNFDRoleDegraded                       = "NFDRoleDegraded"
	conditionNFDRoleBindingDegraded                = "NFDRoleBindingDegraded"
	conditionNFDClusterRoleDegraded                = "NFDClusterRoleDegraded"
	conditionNFDClusterRoleBindingDegraded         = "NFDClusterRoleBindingDegraded"
	conditionNFDSecurityContextConstraintsDegraded = "NFDSecurityContextConstraintsDegraded"

	// Unknown errors. (Catch all)
	errorNFDWorkerDaemonSetUnknown = "NFDWorkerDaemonSetCorrupted"
	errorNFDMasterDaemonSetUnknown = "NFDMasterDaemonSetCorrupted"

	// Invalid node type. (Denotes that the node should be either
	// 'worker' or 'master')
	errorInvalidNodeType = "InvalidNodeTypeSelected"

	// More nodes are listed as "ready" than selected
	errorTooManyNFDWorkerDaemonSetReadyNodes = "NFDWorkerDaemonSetHasMoreNodesThanScheduled"
	errorTooManyNFDMasterDaemonSetReadyNodes = "NFDMasterDaemonSetHasMoreNodesThanScheduled"

	// DaemonSet warnings (for "Progressing" conditions)
	warningNumberOfReadyNodesIsLessThanScheduled = "warningNumberOfReadyNodesIsLessThanScheduled"
	warningNFDWorkerDaemonSetProgressing         = "warningNFDWorkerDaemonSetProgressing"
	warningNFDMasterDaemonSetProgressing         = "warningNFDMasterDaemonSetProgressing"
)

// updateStatus is used to update the status of a resource (e.g., degraded,
// available, etc.)
func (r *NodeFeatureDiscoveryReconciler) updateStatus(nfd *nfdv1.NodeFeatureDiscovery, conditions []conditionsv1.Condition) error {

	// The actual 'nfd' object should *not* be modified when trying to
	// check the object's status. This variable is a dummy variable used
	// to set temporary conditions.
	nfdCopy := nfd.DeepCopy()

	// If a set of conditions exists, then it should be added to the
	// 'nfd' Copy.
	if conditions != nil {
		nfdCopy.Status.Conditions = conditions
	}

	// Next step is to check if we need to update the status
	modified := false

	// Because there are only four possible conditions (degraded, available,
	// updatable, and progressing), it isn't necessary to check if old
	// conditions should be removed.
	for _, newCondition := range nfdCopy.Status.Conditions {
		oldCondition := conditionsv1.FindStatusCondition(nfd.Status.Conditions, newCondition.Type)
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
func (r *NodeFeatureDiscoveryReconciler) updateDegradedCondition(nfd *nfdv1.NodeFeatureDiscovery, condition string, conditionErr error) (ctrl.Result, error) {

	// It is already assumed that the resource has been degraded, so the first
	// step is to gather the correct list of conditions.
	var conditionErrMsg string = "Degraded"
	if conditionErr != nil {
		conditionErrMsg = conditionErr.Error()
	}
	conditions := r.getDegradedConditions(condition, conditionErrMsg)
	if err := r.updateStatus(nfd, conditions); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

// updateProgressingCondition is used to mark a given resource as "progressing" so
// that the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateProgressingCondition(nfd *nfdv1.NodeFeatureDiscovery, condition string, conditionErr error) (ctrl.Result, error) {

	// It is already assumed that the resource is "progressing," so the first
	// step is to gather the correct list of conditions.
	var conditionErrMsg string = "Progressing"
	if conditionErr != nil {
		conditionErrMsg = conditionErr.Error()
	}
	conditions := r.getProgressingConditions(condition, conditionErrMsg)
	if err := r.updateStatus(nfd, conditions); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, nil
}

// getAvailableConditions returns a list of conditionsv1.Condition objects and marks
// every condition as FALSE except for conditionsv1.ConditionAvailable so that the
// reconciler can determine that the resource is available.
func (r *NodeFeatureDiscoveryReconciler) getAvailableConditions() []conditionsv1.Condition {
	now := time.Now()
	return []conditionsv1.Condition{
		{
			Type:               conditionsv1.ConditionAvailable,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
	}
}

// getDegradedConditions returns a list of conditionsv1.Condition objects and marks
// every condition as FALSE except for conditionsv1.ConditionDegraded so that the
// reconciler can determine that the resource is degraded.
func (r *NodeFeatureDiscoveryReconciler) getDegradedConditions(reason string, message string) []conditionsv1.Condition {
	now := time.Now()
	return []conditionsv1.Condition{
		{
			Type:               conditionsv1.ConditionAvailable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
	}
}

// getProgressingConditions returns a list of conditionsv1.Condition objects and marks
// every condition as FALSE except for conditionsv1.ConditionProgressing so that the
// reconciler can determine that the resource is progressing.
func (r *NodeFeatureDiscoveryReconciler) getProgressingConditions(reason string, message string) []conditionsv1.Condition {
	now := time.Now()
	return []conditionsv1.Condition{
		{
			Type:               conditionsv1.ConditionAvailable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
	}
}

// The status of the resource (available, upgradeable, progressing, or
// degraded).
type Status struct {

	// Is the resource available, upgradable, etc.?
	isAvailable   bool
	isUpgradeable bool
	isProgressing bool
	isDegraded    bool
}

// initializeDegradedStatus initializes the status struct to degraded
func initializeDegradedStatus() Status {
	return Status{
		isAvailable:   false,
		isUpgradeable: false,
		isProgressing: false,
		isDegraded:    true,
	}
}

// getWorkerDaemonSetConditions is a wrapper around "getDaemonSetConditions" for
// worker DaemonSets
func (r *NodeFeatureDiscoveryReconciler) getWorkerDaemonSetConditions(ctx context.Context) (Status, error) {
	return r.getDaemonSetConditions(ctx, worker)
}

// getMasterDaemonSetConditions is a wrapper around "getDaemonSetConditions" for
// master DaemonSets
func (r *NodeFeatureDiscoveryReconciler) getMasterDaemonSetConditions(ctx context.Context) (Status, error) {
	return r.getDaemonSetConditions(ctx, master)
}

// getDaemonSetConditions gets the current status of a DaemonSet. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getDaemonSetConditions(ctx context.Context, node nodeType) (Status, error) {

	// Initialize the resource's status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing DaemonSet from the reconciler
	var nodeName string
	if node == worker {
		nodeName = workerName
	} else if node == master {
		nodeName = masterName
	} else {
		return status, errors.New(errorInvalidNodeType)
	}

	ds, err := r.getDaemonSet(ctx, nfdNamespace, nodeName)
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
		if node == worker {
			return status, errors.New(errorNFDWorkerDaemonSetUnknown)
		}
		return status, errors.New(errorNFDMasterDaemonSetUnknown)
	}
	if numberUnavailable > 0 {
		status.isProgressing = true
		status.isDegraded = false
		if node == worker {
			return status, errors.New(warningNFDWorkerDaemonSetProgressing)
		}
		return status, errors.New(warningNFDMasterDaemonSetProgressing)
	}

	// If there are none scheduled, then we have a problem because we should
	// at least see 1 pod per node, even after the scheduling happens.
	if currentNumberScheduled == 0 {
		if node == worker {
			return status, errors.New(conditionNFDWorkerDaemonSetDegraded)
		}
		return status, errors.New(conditionNFDMasterDaemonSetDegraded)
	}

	// Just check in case the number of "ready" nodes is greater than the
	// number of scheduled ones (for whatever reason)
	if numberReady > currentNumberScheduled {
		status.isDegraded = false
		if node == worker {
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
func (r *NodeFeatureDiscoveryReconciler) getServiceConditions(ctx context.Context) (Status, error) {

	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing Service from the reconciler
	_, err := r.getService(ctx, nfdNamespace, masterName)

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
func (r *NodeFeatureDiscoveryReconciler) getRoleConditions(ctx context.Context) (Status, error) {

	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing Role from the reconciler
	_, err := r.getRole(ctx, nfdNamespace, workerName)

	// If the error is not nil, then the Role hasn't been (re)created yet
	if err != nil {
		return status, errors.New(conditionNFDRoleDegraded)
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}

// getDaemonRoleBindingConditions gets the current status of a RoleBinding. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getRoleBindingConditions(ctx context.Context) (Status, error) {

	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing RoleBinding from the reconciler
	_, err := r.getRoleBinding(ctx, nfdNamespace, workerName)

	// If the error is not nil, then the RoleBinding hasn't been (re)created yet
	if err != nil {
		return status, errors.New(conditionNFDRoleBindingDegraded)
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}

// geClusterRoleConditions gets the current status of a ClusterRole. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getClusterRoleConditions(ctx context.Context) (Status, error) {

	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing ClusterRole from the reconciler
	_, err := r.getClusterRole(ctx, "", masterName)

	// If 'clusterRole' is nil, then it hasn't been (re)created yet
	if err != nil {
		return status, errors.New(conditionNFDClusterRoleDegraded)
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}

// getClusterRoleBindingConditions gets the current status of a ClusterRoleBinding.
// If an error occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBindingConditions(ctx context.Context) (Status, error) {

	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing ClusterRoleBinding from the reconciler
	_, err := r.getClusterRoleBinding(ctx, "", masterName)

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
// worker service accounts
func (r *NodeFeatureDiscoveryReconciler) getWorkerServiceAccountConditions(ctx context.Context) (Status, error) {
	return r.getServiceAccountConditions(ctx, worker)
}

// getMasterServiceAccountConditions is a wrapper around "getServiceAccountConditions" for
// master service accounts
func (r *NodeFeatureDiscoveryReconciler) getMasterServiceAccountConditions(ctx context.Context) (Status, error) {
	return r.getServiceAccountConditions(ctx, master)
}

// getServiceAccountConditions gets the current status of a ServiceAccount. If an error
// occurs, this function returns the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getServiceAccountConditions(ctx context.Context, node nodeType) (Status, error) {

	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing ServiceAccount from the reconciler
	var nodeName string
	if node == worker {
		nodeName = workerName
	} else if node == master {
		nodeName = masterName
	} else {
		return status, errors.New(errorInvalidNodeType)
	}

	// Get the service account from the reconciler
	_, err := r.getServiceAccount(ctx, nfdNamespace, nodeName)

	// If the error is not nil, then the ServiceAccount hasn't been (re)created yet
	if err != nil {
		if node == worker {
			return status, errors.New(conditionNFDWorkerServiceAccountDegraded)
		}
		return status, errors.New(conditionNFDMasterServiceAccountDegraded)
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}

// getSecurityContextConstraints gets the current status of a
// SecurityContextConstraints. If an error occurs, this function returns
// the corresponding error message
func (r *NodeFeatureDiscoveryReconciler) getSecurityContextConstraintsConditions(ctx context.Context) (Status, error) {

	// Initialize status to 'Degraded'
	status := initializeDegradedStatus()

	// Get the existing SecurityContextConstraints from the reconciler
	_, err := r.getSecurityContextConstraints(ctx, nfdNamespace, masterName)

	// If the error is not nil, then the SecurityContextConstraints
	// hasn't been (re)created yet
	if err != nil {
		return status, errors.New(conditionNFDSecurityContextConstraintsDegraded)
	}

	// Set the resource to available
	status.isAvailable = true
	status.isDegraded = false

	return status, nil
}
