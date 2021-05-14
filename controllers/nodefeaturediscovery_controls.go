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

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type controlFunc []func(n NFD) (ResourceStatus, error)

// ResourceStatus defines the status of the resource as being
// Ready (=0) or NotReady (=1)
type ResourceStatus int

const (
	Ready ResourceStatus = iota
	NotReady

	defaultServicePort int = 12000
)

// String returns the status of the resource as being Ready,
// NotReady, or Unknown Resource Status
func (s ResourceStatus) String() string {
	names := [...]string{
		"Ready",
		"NotReady"}

	// Ideally, 's' should be either Ready (=0) or NotReady (=1), but 
	// we may run into a case where we get an unknown status, so return
	// information stating that the resource status is unknown
	if s < Ready || s > NotReady {
		return "Unkown Resources Status"
	}
	return names[s]
}

// Namespace checks if the Namespace for NFD exists and attempts to
// create it if it doesn't exist
func Namespace(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// Namespace object, so let's get the resource's Namespace object
	obj := n.resources[state].Namespace

	// found states if the Namespace was found
	found := &corev1.Namespace{}
	logger := log.WithValues("Namespace", obj.Name, "Namespace", "Cluster")

	// Look for the Namespace to see if it exists, and if so, check if
	// it's Ready/NotReady. If the Namespace does not exist, then 
	// attempt to create it
	logger.Info("Looking for")
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating ")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found, skipping update")

	return Ready, nil
}

// ServiceAccount checks if the ServiceAccount for NFD exists and attempts to
// create it if it doesn't exist.
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
	logger := log.WithValues("ServiceAccount", obj.Name, "Namespace", obj.Namespace)

	logger.Info("Looking for")

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
		logger.Info("Not found, creating ")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found, skipping update")

	return Ready, nil
}

// ClusterRole attempts to create a ClusterRole in a given Namespace.
// If the ClusterRole already exists, then attempt to update it.
func ClusterRole(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ClusterRole object, so let's get the resource's ClusterRole
	// object
	obj := n.resources[state].ClusterRole

	// found states if the ClusterRole was found
	found := &rbacv1.ClusterRole{}
	logger := log.WithValues("ClusterRole", obj.Name, "Namespace", obj.Namespace)

	logger.Info("Looking for")

	// Look for the ClusterRole to see if it exists, and if so, check
	// if it's Ready/NotReady. If the ClusterRole does not exist, then 
	// attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the ClusterRole, let's attempt to update it
	logger.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// ClusterRoleBinding attempts to create a ClusterRoleBinding in a given
// Namespace. If the ClusterRoleBinding already exists, then attempt to
// update it.
func ClusterRoleBinding(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ClusterRoleBinding object, so let's get the resource's
	// ClusterRoleBinding object
	obj := n.resources[state].ClusterRoleBinding

	// found states if the ClusterRoleBinding was found
	found := &rbacv1.ClusterRoleBinding{}
	logger := log.WithValues("ClusterRoleBinding", obj.Name, "Namespace", obj.Namespace)

	// It is also assumed that our ClusterRoleBinding has a defined
	// Namespace
	obj.Subjects[0].Namespace = n.ins.GetNamespace()

	logger.Info("Looking for")

	// Look for the ClusterRoleBinding to see if it exists, and if so,
	// check if it's Ready/NotReady. If the ClusterRoleBinding does not
	// exist, then attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the ClusterRoleBinding, let's attempt to update it
	logger.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// Role attempts to create a Role in a given Namespace. If the Role
// already exists, then attempt to update it.
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
	logger := log.WithValues("Role", obj.Name, "Namespace", obj.Namespace)

	logger.Info("Looking for")

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
		logger.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the Role, let's attempt to update it
	logger.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// RoleBinding attempts to create a RoleBinding in a given Namespace. If
// the RoleBinding already exists, then attempt to update it.
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
	logger := log.WithValues("RoleBinding", obj.Name, "Namespace", obj.Namespace)

	logger.Info("Looking for")

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
		logger.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the RoleBinding, let's attempt to update it
	logger.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// ConfigMap attempts to create a ConfigMap in a given Namespace. If
// the ConfigMap already exists, then attempt to update it.
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
	logger := log.WithValues("ConfigMap", obj.Name, "Namespace", obj.Namespace)

	logger.Info("Looking for")

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
		logger.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the ConfigMap, let's attempt to update it
	logger.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// DaemonSet attempts to create a DaemonSet in a given Namespace. If
// the DaemonSet already exists, then attempt to update it.
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
	logger := log.WithValues("DaemonSet", obj.Name, "Namespace", obj.Namespace)

	logger.Info("Looking for")

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
		logger.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the DaemonSet, let's attempt to update it
	logger.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// Service attempts to create a Service in a given Namespace. If the
// Service already exists, then attempt to update it.
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
	logger := log.WithValues("Service", obj.Name, "Namespace", obj.Namespace)

	logger.Info("Looking for")

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
		logger.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found, updating")


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

// SecurityContextConstraints attempts to create SecurityContextConstraints
// in a given Namespace. If the scc already exists, then attempt to update it.
func SecurityContextConstraints(n NFD) (ResourceStatus, error) {

	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be an
	// scc object, so let's get the resource's scc object
	obj := n.resources[state].SecurityContextConstraints

	// Set the correct namespace for SCC when installed in non default namespace
	obj.Users[0] = "system:serviceaccount:" + n.ins.GetNamespace() + ":" + obj.GetName()

	// found states if the scc was found
	found := &secv1.SecurityContextConstraints{}
	logger := log.WithValues("SecurityContextConstraints", obj.Name, "Namespace", "default")

	logger.Info("Looking for")

	// Look for the scc to see if it exists, and if so, check if it's
	// Ready/NotReady. If the scc does not exist, then attempt to create
	// it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create", "Error", err)
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found, updating")

	// If we found the scc, let's attempt to update it with the resource
	// version we found
	required := obj.DeepCopy()
	required.ResourceVersion = found.ResourceVersion

	err = n.rec.Client.Update(context.TODO(), required)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}
