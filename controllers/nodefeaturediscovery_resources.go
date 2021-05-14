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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/kubectl/pkg/scheme"
)

// assetsFromFile is a list where each item in the list contains the
// contents of a given file as a list of bytes
type assetsFromFile []byte

// Resources holds objects owned by NFD. This struct is used with the
// 'NFD' struct to assist in the process of checking if NFD's resources
// are 'Ready' or 'NotReady'.
type Resources struct {
	Namespace                  corev1.Namespace
	ServiceAccount             corev1.ServiceAccount
	Role                       rbacv1.Role
	RoleBinding                rbacv1.RoleBinding
	ClusterRole                rbacv1.ClusterRole
	ClusterRoleBinding         rbacv1.ClusterRoleBinding
	ConfigMap                  corev1.ConfigMap
	DaemonSet                  appsv1.DaemonSet
	Pod                        corev1.Pod
	Service                    corev1.Service
	SecurityContextConstraints secv1.SecurityContextConstraints
}

// Add3dpartyResourcesToScheme Adds 3rd party resources To the operator
func Add3dpartyResourcesToScheme(scheme *runtime.Scheme) error {

	if err := secv1.AddToScheme(scheme); err != nil {
		return err
	}
	return nil
}

// filePathWalkDir takes a path as an input and finds all files
// in that path, but not directories
func filePathWalkDir(root string) ([]string, error) {

	// files contains the list of files found in the path
	// 'root'
	var files []string

	// Walk through the files in 'path', and if the os.FileInfo object
	// states that the item is not a directory, append it to the list
	// of files
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// getAssetsFrom takes a path as an input and grabs all of the
// file names in that path, then returns a list of the manifests
// it found in that path.
func getAssetsFrom(path string) []assetsFromFile {

	// manifests is a list type where each item in the list
	// contains the contents of a given asset (manifest)
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

		// Read the file and return its contents in 'buffer'
		buffer, err := ioutil.ReadFile(file)

		// If we have an error, then something unexpectedly went
		// wrong when reading the file's contents
		if err != nil {
			panic(err)
		}

		// If the reading goes smoothly, then append the buffer
		// (the file's contents) to the list of manifests
		manifests = append(manifests, buffer)
	}
	return manifests
}


func addResourcesControls(path string) (Resources, controlFunc) {

	// res is a Resources object that contains information about
	// a given manifest, such as the Namespace and ServiceAccount
	// being used
	res := Resources{}

	// ctrl is a controlFunc object that contains a function
	// that returns information about the status of a resource
	// (i.e., Ready or NotReady)
	ctrl := controlFunc{}

	// Get the list of manifests from the given path
	manifests := getAssetsFrom(path)

	// s and reg are used later on to parse the manifest YAML
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
		scheme.Scheme)
	reg, _ := regexp.Compile(`\b(\w*kind:\w*)\B.*\b`)

	// For each manifest, find its kind, then append the appropriate
	// function (e.g., 'Namespace' or 'Role') to ctrl so that the
	// Namespace, Role, etc. can be parsed
	for _, m := range manifests {
		kind := reg.FindString(string(m))
		slce := strings.Split(kind, ":")
		kind = strings.TrimSpace(slce[1])

		switch kind {
		case "Namespace":
			_, _, err := s.Decode(m, nil, &res.Namespace)
			panicIfError(err)
			ctrl = append(ctrl, Namespace)
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
		case "Service":
			_, _, err := s.Decode(m, nil, &res.Service)
			panicIfError(err)
			ctrl = append(ctrl, Service)
		case "SecurityContextConstraints":
			_, _, err := s.Decode(m, nil, &res.SecurityContextConstraints)
			panicIfError(err)
			ctrl = append(ctrl, SecurityContextConstraints)

		default:
			log.Info("Unknown Resource: ", "Kind", kind)
		}

	}

	return res, ctrl
}

// Trigger a panic if an error occurs
func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
