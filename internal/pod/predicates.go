package pod

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func IgnoreDeletionPredicate() predicate.Predicate {
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

func HasAnnotations(obj client.Object) bool {
	_, ok := obj.GetAnnotations()[ReceiverAnnotation]
	if ok {
		return true
	}
	_, ok = obj.GetAnnotations()[SenderAnnotation]
	if ok {
		return true
	}
	return false
}
