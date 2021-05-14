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
	"errors"

	nfdv1 "github.com/kubernetes-sigs/node-feature-discovery-operator/api/v1"
)

// NFD holds the needed information to watch from the Controller. The
// following descriptions elaborate on each field in this struct:
type NFD struct {

	// resources contains information about NFD's resources. For more
	// information, see ./nodefeaturediscovery_resources.go
	resources []Resources

	// controls is a list that contains the status of an NFD resource
	// as being Ready (=0) or NotReady (=1)
	controls []controlFunc

	// rec represents the NFD reconciler struct used for reconciliation
	rec *NodeFeatureDiscoveryReconciler

	// ins is the NodeFeatureDiscovery struct that contains the Schema
	// for the nodefeaturediscoveries API
	ins *nfdv1.NodeFeatureDiscovery

	// idx is the index that is used to step through the 'controls' list
	// and is set to 0 upon calling 'init()'
	idx int
}

// addState takes a given path and finds resources in that path, then
// appends a list of ctrl's functions to the NFD object's 'controls'
// field and adds the list of resources found to 'n.resources'
func (n *NFD) addState(path string) {
	res, ctrl := addResourcesControls(path)
	n.controls = append(n.controls, ctrl)
	n.resources = append(n.resources, res)
}

// init initializes an NFD object by populating the fields before
// attempting to run any kind of check.
func (n *NFD) init(
	r *NodeFeatureDiscoveryReconciler,
	i *nfdv1.NodeFeatureDiscovery,
) {
	n.rec = r
	n.ins = i
	n.idx = 0
	if len(n.controls) == 0 {
		n.addState("/opt/nfd/master")
		n.addState("/opt/nfd/worker")
	}
}

// step steps through the list of functions stored in 'n.controls',
// then attempts to determine if the given resource is Ready or
// NotReady. (See the following file for a list of functions that
// 'n.controls' can take on: ./nodefeaturediscovery_resources.go.)
func (n *NFD) step() error {

	// For each function in n.controls, attempt to check the status of
	// the relevant resource. If no error occurs and the resource is
	// defined as being "NotReady," then return an error saying it's not
	// ready. Otherwise, return the status as being ready, then increment
	// the index for n.controls so that we can parse the next resource.
	for _, fs := range n.controls[n.idx] {
		stat, err := fs(*n)
		if err != nil {
			return err
		}
		if stat != Ready {
			return errors.New("ResourceNotReady")
		}
	}
	n.idx = n.idx + 1
	return nil
}

// last checks if the last index equals the number of functions
// stored in n.controls.
func (n *NFD) last() bool {
	return n.idx == len(n.controls)
}

//func (r *NodeFeatureDiscoveryReconciler) updateStatus(cr *nfdv1.NodeFeatureDiscovery, conditions []conditionsv1.Condition) error {
//	customResourceCopy := cr.DeepCopy()
//
//	if conditions != nil {
//		customResourceCopy.Status.Conditions = conditions
//	}
//
//	// check if we need to update the status
//	modified := false
//
//	// since we always set the same four conditions, we don't need to check if we need to remove old conditions
//	for _, newCondition := range customResourceCopy.Status.Conditions {
//		oldCondition := conditionsv1.FindStatusCondition(cr.Status.Conditions, newCondition.Type)
//		if oldCondition == nil {
//			modified = true
//			break
//		}
//
//		// ignore timestamps to avoid infinite reconcile loops
//		if oldCondition.Status != newCondition.Status ||
//			oldCondition.Reason != newCondition.Reason ||
//			oldCondition.Message != newCondition.Message {
//
//			modified = true
//			break
//		}
//	}
//
//	if !modified {
//		return nil
//	}
//
//	klog.Infof("Updating the nodeFeatureDiscovery %q status", cr.Name)
//	return r.Status().Update(context.TODO(), customResourceCopy)
//}
//
//func (r *NodeFeatureDiscoveryReconciler) getAvailableConditions() []conditionsv1.Condition {
//	now := time.Now()
//	return []conditionsv1.Condition{
//		{
//			Type:               conditionsv1.ConditionAvailable,
//			Status:             corev1.ConditionTrue,
//			LastTransitionTime: metav1.Time{Time: now},
//			LastHeartbeatTime:  metav1.Time{Time: now},
//		},
//		{
//			Type:               conditionsv1.ConditionUpgradeable,
//			Status:             corev1.ConditionTrue,
//			LastTransitionTime: metav1.Time{Time: now},
//			LastHeartbeatTime:  metav1.Time{Time: now},
//		},
//		{
//			Type:               conditionsv1.ConditionProgressing,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: metav1.Time{Time: now},
//			LastHeartbeatTime:  metav1.Time{Time: now},
//		},
//		{
//			Type:               conditionsv1.ConditionDegraded,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: metav1.Time{Time: now},
//			LastHeartbeatTime:  metav1.Time{Time: now},
//		},
//	}
//}
//
//func (r *NodeFeatureDiscoveryReconciler) getDegradedConditions(reason string, message string) []conditionsv1.Condition {
//	now := time.Now()
//	return []conditionsv1.Condition{
//		{
//			Type:               conditionsv1.ConditionAvailable,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: metav1.Time{Time: now},
//			LastHeartbeatTime:  metav1.Time{Time: now},
//		},
//		{
//			Type:               conditionsv1.ConditionUpgradeable,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: metav1.Time{Time: now},
//			LastHeartbeatTime:  metav1.Time{Time: now},
//		},
//		{
//			Type:               conditionsv1.ConditionProgressing,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: metav1.Time{Time: now},
//			LastHeartbeatTime:  metav1.Time{Time: now},
//		},
//		{
//			Type:               conditionsv1.ConditionDegraded,
//			Status:             corev1.ConditionTrue,
//			LastTransitionTime: metav1.Time{Time: now},
//			LastHeartbeatTime:  metav1.Time{Time: now},
//			Reason:             reason,
//			Message:            message,
//		},
//	}
//}
//
//func (r *NodeFeatureDiscoveryReconciler) getProgressingConditions(reason string, message string) []conditionsv1.Condition {
//	now := time.Now()
//
//	return []conditionsv1.Condition{
//		{
//			Type:               conditionsv1.ConditionAvailable,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: metav1.Time{Time: now},
//		},
//		{
//			Type:               conditionsv1.ConditionUpgradeable,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: metav1.Time{Time: now},
//		},
//		{
//			Type:               conditionsv1.ConditionProgressing,
//			Status:             corev1.ConditionTrue,
//			LastTransitionTime: metav1.Time{Time: now},
//			Reason:             reason,
//			Message:            message,
//		},
//		{
//			Type:               conditionsv1.ConditionDegraded,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: metav1.Time{Time: now},
//		},
//	}
//}
//
