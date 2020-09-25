/*
Copyright 2020 The Kubernetes Authors.

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

package config

import (
	"os"
)

const (
	nodeFeautreDiscoveryImageDefault string = "k8s.gcr.io/nfd/node-feature-discovery:v0.6.0"
)

// NodeFeatureDiscoveryImage returns the operator's operand/nfd image path.
func NodeFeatureDiscoveryImage() string {
	nodeFeatureDiscoveryImage := os.Getenv("NODE_FEATURE_DISCOVERY_IMAGE")

	if len(nodeFeatureDiscoveryImage) > 0 {
		return nodeFeatureDiscoveryImage
	}

	return nodeFeautreDiscoveryImageDefault
}
