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

// NFD holds the needed information to watch from the Controller
type NFD struct {
	resources []Resources
	controls  []controlFunc
	rec       *NodeFeatureDiscoveryReconciler
	ins       *nfdv1.NodeFeatureDiscovery
	idx       int
}

func (n *NFD) addState(path string) {
	res, ctrl := addResourcesControls(path)
	n.controls = append(n.controls, ctrl)
	n.resources = append(n.resources, res)
}

func (n *NFD) init(
	r *NodeFeatureDiscoveryReconciler,
	i *nfdv1.NodeFeatureDiscovery,
) {
	n.rec = r
	n.ins = i
	n.idx = 0
	n.addState("/opt/nfd/master")
	n.addState("/opt/nfd/worker")
}

func (n *NFD) step() error {
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
