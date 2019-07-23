package controller

import (
	"github.com/copybird/copybird-operator/pkg/controller/copybird"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, copybird.Add)
}
