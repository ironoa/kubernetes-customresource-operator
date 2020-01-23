package controller

import (
	"github.com/ironoa/kubernetes-customresource-operator/pkg/controller/customresource"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, customresource.Add)
}
