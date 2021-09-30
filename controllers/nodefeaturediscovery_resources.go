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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/kubectl/pkg/scheme"
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
		buffer, err := ioutil.ReadFile(file)
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

		default:
			log.Info("Unknown Resource: ", "Kind", kind)
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
