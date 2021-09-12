/*
Copyright 2021.

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

package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// OnlyIncludeAnnotatedPredicate is a predicate that is passed to the reconciler
// to only watch pods with the required annotations
func OnlyIncludeAnnotatedPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			value := HasAnnotations(e.ObjectOld) || HasAnnotations(e.ObjectNew)
			return value
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return HasAnnotations(e.Object)
		},
		CreateFunc: func(ce event.CreateEvent) bool {
			return HasAnnotations(ce.Object)
		},
	}
}

// HasAnnotations checks whether the passed object has one of the
// sender or receiver annotations
func HasAnnotations(obj client.Object) bool {
	_, ok := obj.GetAnnotations()[ReceiverAnnotation]
	if ok {
		return true
	}
	_, ok = obj.GetAnnotations()[SenderAnnotation]
	return ok
}
