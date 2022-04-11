/*
Copyright 2022 The Kubernetes Authors.

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

package version

import (
	"runtime/debug"
	"strings"
)

const undefinedVersion string = "undefined"

// Must not be const, supposed to be set using ldflags at build time
var version = undefinedVersion

// Get returns the version as a string
func Get() string {
	return version
}

func GetWithVCSRevision(v string, bi *debug.BuildInfo) string {
	var (
		vcsRevision string
		vcsDirty    = false
	)

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			vcsRevision = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				vcsDirty = true
			}
		}
	}

	var sb strings.Builder

	sb.WriteString(v)

	if vcsRevision == "" {
		vcsRevision = "undefined"
	}

	sb.WriteRune('-')
	sb.WriteString(vcsRevision)

	if vcsDirty {
		sb.WriteRune('-')
		sb.WriteString("dirty")
	}

	return sb.String()
}

// Undefined returns if version is at it's default value
func Undefined() bool {
	return version == undefinedVersion
}
