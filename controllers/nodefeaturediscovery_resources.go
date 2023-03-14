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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// assetsFromFile is the content of an asset file as raw data
type assetsFromFile []byte

// Resources holds objects owned by NFD
type Resources struct {
	Namespace          corev1.Namespace
	ServiceAccount     corev1.ServiceAccount
	Role               rbacv1.Role
	RoleBinding        rbacv1.RoleBinding
	ClusterRole        rbacv1.ClusterRole
	ClusterRoleBinding rbacv1.ClusterRoleBinding
	ConfigMap          corev1.ConfigMap
	DaemonSet          appsv1.DaemonSet
	Job                batchv1.Job
	Deployment         appsv1.Deployment
	Pod                corev1.Pod
	Service            corev1.Service
}

// filePathWalkDir finds all non-directory files under the given path recursively,
// i.e. including its subdirectories
func filePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// getAssetsFrom recursively reads all manifest files under a given path
func getAssetsFrom(path string) []assetsFromFile {
	// All assets (manifests) as raw data
	manifests := []assetsFromFile{}
	assets := path

	// For the given path, find a list of all the files
	files, err := filePathWalkDir(assets)
	if err != nil {
		panic(err)
	}

	// For each file in the 'files' list, read the file
	// and store its contents in 'manifests'
	for _, file := range files {
		buffer, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}

		manifests = append(manifests, buffer)
	}
	return manifests
}

func addResourcesControls(path string) (Resources, controlFunc) {
	// Information about the manifest
	res := Resources{}

	// A list of control functions for checking the status of a resource
	ctrl := controlFunc{}

	// Get the list of manifests from the given path
	manifests := getAssetsFrom(path)

	// s and reg are used later on to parse the manifest YAML
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
		scheme.Scheme)
	reg, _ := regexp.Compile(`\b(\w*kind:\w*)\B.*\b`)

	// Append the appropriate control function depending on the kind
	for _, m := range manifests {
		kind := reg.FindString(string(m))
		slce := strings.Split(kind, ":")
		kind = strings.TrimSpace(slce[1])

		switch kind {
		case "ServiceAccount":
			_, _, err := s.Decode(m, nil, &res.ServiceAccount)
			panicIfError(err)
			ctrl = append(ctrl, ServiceAccount)
		case "ClusterRole":
			_, _, err := s.Decode(m, nil, &res.ClusterRole)
			panicIfError(err)
			ctrl = append(ctrl, ClusterRole)
		case "ClusterRoleBinding":
			_, _, err := s.Decode(m, nil, &res.ClusterRoleBinding)
			panicIfError(err)
			ctrl = append(ctrl, ClusterRoleBinding)
		case "Role":
			_, _, err := s.Decode(m, nil, &res.Role)
			panicIfError(err)
			ctrl = append(ctrl, Role)
		case "RoleBinding":
			_, _, err := s.Decode(m, nil, &res.RoleBinding)
			panicIfError(err)
			ctrl = append(ctrl, RoleBinding)
		case "ConfigMap":
			_, _, err := s.Decode(m, nil, &res.ConfigMap)
			panicIfError(err)
			ctrl = append(ctrl, ConfigMap)
		case "DaemonSet":
			_, _, err := s.Decode(m, nil, &res.DaemonSet)
			panicIfError(err)
			ctrl = append(ctrl, DaemonSet)
		case "Deployment":
			_, _, err := s.Decode(m, nil, &res.Deployment)
			panicIfError(err)
			ctrl = append(ctrl, Deployment)
		case "Job":
			_, _, err := s.Decode(m, nil, &res.Job)
			panicIfError(err)
			ctrl = append(ctrl, Job)
		case "Service":
			_, _, err := s.Decode(m, nil, &res.Service)
			panicIfError(err)
			ctrl = append(ctrl, Service)

		default:
			klog.Info("Unknown Resource: ", "Kind", kind)
		}
	}

	return res, ctrl
}

// panicIfError panics in case of an error
func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

// getServiceAccount gets one of the NFD Operand's ServiceAccounts
func (r *NodeFeatureDiscoveryReconciler) getServiceAccount(ctx context.Context, namespace string, name string) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, sa)
	return sa, err
}

// getDaemonSet gets one of the NFD Operand's DaemonSets
func (r *NodeFeatureDiscoveryReconciler) getDaemonSet(ctx context.Context, namespace string, name string) (*appsv1.DaemonSet, error) {
	ds := &appsv1.DaemonSet{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, ds)
	return ds, err
}

// getDeployment gets one of the NFD Operand's Deployment
func (r *NodeFeatureDiscoveryReconciler) getDeployment(ctx context.Context, namespace string, name string) (*appsv1.Deployment, error) {
	d := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, d)
	return d, err
}

// getJob gets one of the NFD Operand's Job
func (r *NodeFeatureDiscoveryReconciler) getJob(ctx context.Context, namespace string, name string) (*batchv1.Job, error) {
	j := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, j)
	return j, err
}

// getConfigMap gets one of the NFD Operand's ConfigMap
func (r *NodeFeatureDiscoveryReconciler) getConfigMap(ctx context.Context, namespace string, name string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cm)
	return cm, err
}

// getService gets one of the NFD Operand's Services
func (r *NodeFeatureDiscoveryReconciler) getService(ctx context.Context, namespace string, name string) (*corev1.Service, error) {
	svc := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, svc)
	return svc, err
}

// getRole gets one of the NFD Operand's Roles
func (r *NodeFeatureDiscoveryReconciler) getRole(ctx context.Context, namespace string, name string) (*rbacv1.Role, error) {
	role := &rbacv1.Role{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, role)
	return role, err
}

// getRoleBinding gets one of the NFD Operand's RoleBindings
func (r *NodeFeatureDiscoveryReconciler) getRoleBinding(ctx context.Context, namespace string, name string) (*rbacv1.RoleBinding, error) {
	rb := &rbacv1.RoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, rb)
	return rb, err
}

// getClusterRole gets one of the NFD Operand's ClusterRoles
func (r *NodeFeatureDiscoveryReconciler) getClusterRole(ctx context.Context, namespace string, name string) (*rbacv1.ClusterRole, error) {
	cr := &rbacv1.ClusterRole{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cr)
	return cr, err
}

// getClusterRoleBinding gets one of the NFD Operand's ClusterRoleBindings
func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBinding(ctx context.Context, namespace string, name string) (*rbacv1.ClusterRoleBinding, error) {
	crb := &rbacv1.ClusterRoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, crb)
	return crb, err
}

// deleteServiceAccount deletes one of the NFD Operand's ServiceAccounts
func (r *NodeFeatureDiscoveryReconciler) deleteServiceAccount(ctx context.Context, namespace string, name string) error {
	sa, err := r.getServiceAccount(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), sa)
}

// deleteConfigMap deletes the NFD Operand ConfigMap
func (r *NodeFeatureDiscoveryReconciler) deleteConfigMap(ctx context.Context, namespace string, name string) error {
	cm, err := r.getConfigMap(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), cm)
}

// deleteDaemonSet deletes Operand DaemonSet
func (r *NodeFeatureDiscoveryReconciler) deleteDaemonSet(ctx context.Context, namespace string, name string) error {
	ds, err := r.getDaemonSet(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), ds)
}

// deleteDeployment deletes Operand Deployment
func (r *NodeFeatureDiscoveryReconciler) deleteDeployment(ctx context.Context, namespace string, name string) error {
	d, err := r.getDeployment(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), d)
}

// deleteJob deletes Operand job
func (r *NodeFeatureDiscoveryReconciler) deleteJob(ctx context.Context, namespace string, name string) error {
	j, err := r.getJob(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), j)
}

// deleteService deletes the NFD Operand's Service
func (r *NodeFeatureDiscoveryReconciler) deleteService(ctx context.Context, namespace string, name string) error {
	svc, err := r.getService(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), svc)
}

// deleteRole deletes one of the NFD Operand's Roles
func (r *NodeFeatureDiscoveryReconciler) deleteRole(ctx context.Context, namespace string, name string) error {
	role, err := r.getRole(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), role)
}

// deleteRoleBinding deletes one of the NFD Operand's RoleBindings
func (r *NodeFeatureDiscoveryReconciler) deleteRoleBinding(ctx context.Context, namespace string, name string) error {
	rb, err := r.getRoleBinding(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), rb)
}

// deleteClusterRole deletes one of the NFD Operand's ClusterRoles
func (r *NodeFeatureDiscoveryReconciler) deleteClusterRole(ctx context.Context, namespace string, name string) error {
	cr, err := r.getClusterRole(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), cr)
}

// deleteClusterRoleBinding deletes one of the NFD Operand's ClusterRoleBindings
func (r *NodeFeatureDiscoveryReconciler) deleteClusterRoleBinding(ctx context.Context, namespace string, name string) error {
	crb, err := r.getClusterRoleBinding(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), crb)
}
