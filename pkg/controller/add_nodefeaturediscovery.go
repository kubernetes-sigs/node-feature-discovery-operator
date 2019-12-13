package controller

import (
	"github.com/openshift/node-feature-discovery-operator/pkg/controller/nodefeaturediscovery"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, nodefeaturediscovery.Add)
}
