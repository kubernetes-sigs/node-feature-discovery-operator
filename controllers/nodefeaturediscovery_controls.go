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
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Used for logging purposes
var r NodeFeatureDiscoveryReconciler

type controlFunc []func(n NFD) (ResourceStatus, error)

// ResourceStatus defines the status of the resource as being
// Ready or NotReady
type ResourceStatus int

const (
	Ready ResourceStatus = iota
	NotReady

	defaultServicePort int = 12000
)

// String implements the fmt.Stringer interface and returns describes
// ResourceStatus as a string.
func (s ResourceStatus) String() string {
	names := [...]string{
		"Ready",
		"NotReady"}

	if s < Ready || s > NotReady {
		return "Unkown Resources Status"
	}
	return names[s]
}

// Namespace checks if the Namespace for NFD exists and creates it
// if it doesn't exist
func Namespace(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// Namespace object, so let's get the resource's Namespace object
	obj := n.resources[state].Namespace

	// found states if the Namespace was found
	found := &corev1.Namespace{}

	// Look for the Namespace to see if it exists, and if so, check if
	// it's Ready/NotReady. If the Namespace does not exist, then
	// attempt to create it
	r.Log.Info("Looking for Namespace '", obj.Name, "'")
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Not found, creating ")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	r.Log.Info("Found, skipping update")

	return Ready, nil
}

// ServiceAccount checks the readiness of the NFD ServiceAccount and creates it if it doesn't exist
func ServiceAccount(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ServiceAccount object, so let's get the resource's ServiceAccount
	// object
	obj := n.resources[state].ServiceAccount

	// It is also assumed that our service account has a defined Namespace
	obj.SetNamespace(n.ins.GetNamespace())

	// found states if the ServiceAccount was found
	found := &corev1.ServiceAccount{}

	r.Log.Info("Looking for ServiceAccount '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the ServiceAccount to see if it exists, and if so, check if
	// it's Ready/NotReady. If the ServiceAccount does not exist, then
	// attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Not found, creating ")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	r.Log.Info("Found, skipping update")

	return Ready, nil
}

// ClusterRole checks if the ClusterRole exists, and creates it if it doesn't
func ClusterRole(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ClusterRole object, so let's get the resource's ClusterRole
	// object
	obj := n.resources[state].ClusterRole

	// found states if the ClusterRole was found
	found := &rbacv1.ClusterRole{}

	r.Log.Info("Looking for ClusterRole '", obj.Name, "'")

	// Look for the ClusterRole to see if it exists, and if so, check
	// if it's Ready/NotReady. If the ClusterRole does not exist, then
	// attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the ClusterRole, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// ClusterRoleBinding checks if a ClusterRoleBinding exists and creates one if it doesn't
func ClusterRoleBinding(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ClusterRoleBinding object, so let's get the resource's
	// ClusterRoleBinding object
	obj := n.resources[state].ClusterRoleBinding

	// found states if the ClusterRoleBinding was found
	found := &rbacv1.ClusterRoleBinding{}

	// It is also assumed that our ClusterRoleBinding has a defined
	// Namespace
	obj.Subjects[0].Namespace = n.ins.GetNamespace()

	r.Log.Info("Looking for ClusterRoleBinding '", obj.Name, "'")

	// Look for the ClusterRoleBinding to see if it exists, and if so,
	// check if it's Ready/NotReady. If the ClusterRoleBinding does not
	// exist, then attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the ClusterRoleBinding, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// Role checks if a Role exists and creates a Role if it doesn't
func Role(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// Role object, so let's get the resource's Role object
	obj := n.resources[state].Role

	// The Namespace should already be defined, so let's set the
	// namespace to the namespace defined in the Role object
	obj.SetNamespace(n.ins.GetNamespace())

	// found states if the Role was found
	found := &rbacv1.Role{}

	r.Log.Info("Looking for Role '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the Role to see if it exists, and if so, check if it's
	// Ready/NotReady. If the Role does not exist, then attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the Role, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// RoleBinding checks if a RoleBinding exists and creates a RoleBinding if it doesn't
func RoleBinding(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// RoleBinding object, so let's get the resource's RoleBinding
	// object
	obj := n.resources[state].RoleBinding

	// The Namespace should already be defined, so let's set the
	// namespace to the namespace defined in the
	obj.SetNamespace(n.ins.GetNamespace())

	// found states if the RoleBinding was found
	found := &rbacv1.RoleBinding{}

	r.Log.Info("Looking for RoleBinding", obj.Name, "in Namespace", obj.Namespace)

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the RoleBinding to see if it exists, and if so, check if
	// it's Ready/NotReady. If the RoleBinding does not exist, then attempt
	// to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the RoleBinding, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// ConfigMap checks if a ConfigMap exists and creates one if it doesn't
func ConfigMap(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ConfigMap object, so let's get the resource's ConfigMap object
	obj := n.resources[state].ConfigMap

	// The Namespace should already be defined, so let's set the
	// namespace to the namespace defined in the ConfigMap object
	obj.SetNamespace(n.ins.GetNamespace())

	// Update ConfigMap
	obj.ObjectMeta.Name = "nfd-worker"
	obj.Data["nfd-worker-conf"] = n.ins.Spec.WorkerConfig.ConfigData

	// found states if the ConfigMap was found
	found := &corev1.ConfigMap{}

	r.Log.Info("Looking for ConfigMap '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the ConfigMap to see if it exists, and if so, check if it's
	// Ready/NotReady. If the ConfigMap does not exist, then attempt to create
	// it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the ConfigMap, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// DaemonSet checks the readiness of a DaemonSet and creates one if it doesn't exist
func DaemonSet(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// DaemonSet object, so let's get the resource's DaemonSet object
	obj := n.resources[state].DaemonSet

	// Update the NFD operand image
	obj.Spec.Template.Spec.Containers[0].Image = n.ins.Spec.Operand.ImagePath()

	// Update the image pull policy
	if n.ins.Spec.Operand.ImagePullPolicy != "" {
		obj.Spec.Template.Spec.Containers[0].ImagePullPolicy = n.ins.Spec.Operand.ImagePolicy(n.ins.Spec.Operand.ImagePullPolicy)
	}

	// Update nfd-master service port
	if obj.ObjectMeta.Name == "nfd-master" {
		var args []string
		port := defaultServicePort

		// If the operand service port has already been defined,
		// then set "port" to the defined port. Otherwise, it is
		// ok to just use the defaultServicePort value
		if n.ins.Spec.Operand.ServicePort != 0 {
			port = n.ins.Spec.Operand.ServicePort
		}

		// Now that the port has been determined, append it to
		// the list of args
		args = append(args, fmt.Sprintf("--port=%d", port))

		// Check if running as instance. If not, then it is
		// expected that n.ins.Spec.Instance will return ""
		// https://kubernetes-sigs.github.io/node-feature-discovery/v0.8/advanced/master-commandline-reference.html#-instance
		if n.ins.Spec.Instance != "" {
			args = append(args, fmt.Sprintf("--instance=%s", n.ins.Spec.Instance))
		}

		if len(n.ins.Spec.ExtraLabelNs) != 0 {
			args = append(args, fmt.Sprintf("--extra-label-ns=%s", strings.Join(n.ins.Spec.ExtraLabelNs, ",")))
		}

		if len(n.ins.Spec.ResourceLabels) != 0 {
			args = append(args, fmt.Sprintf("--resource-labels=%s", strings.Join(n.ins.Spec.ResourceLabels, ",")))
		}

		if strings.TrimSpace(n.ins.Spec.LabelWhiteList) != "" {
			args = append(args, fmt.Sprintf("--label-whitelist=%s", n.ins.Spec.LabelWhiteList))
		}

		// Set the args based on the port that was determined
		// and the instance that was determined
		obj.Spec.Template.Spec.Containers[0].Args = args
	}

	// Set namespace based on the NFD namespace. (And again,
	// it is assumed that the Namespace has already been
	// determined before this function was called.)
	obj.SetNamespace(n.ins.GetNamespace())

	// found states if the DaemonSet was found
	found := &appsv1.DaemonSet{}

	r.Log.Info("Looking for DaemonSet '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the DaemonSet to see if it exists, and if so, check if it's
	// Ready/NotReady. If the DaemonSet does not exist, then attempt to
	// create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the DaemonSet, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// Service checks if a Service exists and creates one if it doesn't exist
func Service(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// Service object, so let's get the resource's Service object
	obj := n.resources[state].Service

	// Update ports for the Service. If the service port has already
	// been defined, then that value should be used. Otherwise, just
	// use the defaultServicePort's value.
	if n.ins.Spec.Operand.ServicePort != 0 {
		obj.Spec.Ports[0].Port = int32(n.ins.Spec.Operand.ServicePort)
		obj.Spec.Ports[0].TargetPort = intstr.FromInt(n.ins.Spec.Operand.ServicePort)
	} else {
		obj.Spec.Ports[0].Port = int32(defaultServicePort)
		obj.Spec.Ports[0].TargetPort = intstr.FromInt(defaultServicePort)
	}

	// Set namespace based on the NFD namespace. (And again,
	// it is assumed that the Namespace has already been
	// determined before this function was called.)
	obj.SetNamespace(n.ins.GetNamespace())

	// found states if the Service was found
	found := &corev1.Service{}

	r.Log.Info("Looking for Service '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the Service to see if it exists, and if so, check if it's
	// Ready/NotReady. If the Service does not exist, then attempt to create
	// it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	r.Log.Info("Found, updating")

	// Copy the Service object
	required := obj.DeepCopy()

	// Set the resource version based on what we found when searching
	// for the existing Service. Do the same for ClusterIP
	required.ResourceVersion = found.ResourceVersion
	required.Spec.ClusterIP = found.Spec.ClusterIP

	// If we found the Service, let's attempt to update it with the
	// resource version and cluster IP that was just found
	err = n.rec.Client.Update(context.TODO(), required)

	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}
