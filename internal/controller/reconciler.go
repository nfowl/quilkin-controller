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
	"context"
	"errors"
	"strings"

	"github.com/nfowl/quilkin-controller/internal/store"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// QuilkinReconciler contains the required objects to run the Reconcile loop
type QuilkinReconciler struct {
	client client.Client
	logger *zap.SugaredLogger
	store  *store.SotwStore
}

// NewQuilkinReconciler constructs a new QuilkinReconciler struct from the passed arguments
func NewQuilkinReconciler(c client.Client, l *zap.SugaredLogger, s *store.SotwStore) *QuilkinReconciler {
	return &QuilkinReconciler{
		client: c,
		logger: l,
		store:  s,
	}
}

// Reconcile implements the reconciliation logic for all pods that are acting as either a
// quilkin sender/receiver
func (q *QuilkinReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	pod := &corev1.Pod{}
	err := q.client.Get(ctx, req.NamespacedName, pod)
	q.logger.Debugw("Calling reconciler for pod", "name", req.NamespacedName.String())
	if err != nil {
		q.logger.Debug("Failed to decode pod for reconciling")
	}

	// Pod with the required finalizers is being deleted
	if !pod.DeletionTimestamp.IsZero() && containsString(pod.GetFinalizers(), Finalizer) {
		q.logger.Infow("Handling finalizer")
		value, ok := pod.Annotations[ReceiverAnnotation]
		if ok {
			proxyName, _, err := parseReceiveAnnotation(value)
			if err != nil {
				q.logger.Errorw("Error parsing annotation", "annotation", value)
			}
			q.logger.Infow("Removing receiver", "proxy", proxyName, "pod", pod.Name, "ip", pod.Status.PodIP)
			q.store.RemoveReceiver(proxyName, pod.Name)
		}

		// Handle and remove finalizer for sender
		value, ok = pod.Annotations[SenderAnnotation]
		if ok {
			q.logger.Infow("Removing sender", "sender", value, "pod", pod.Name)
			_ = q.store.RemoveSender(value, pod.Name)
			// if lastNode {
			// 	q.logger.Infow("Removing quilkin sender configmap", "configmap", "quilkin-"+value)
			// 	cm := &corev1.ConfigMap{}
			// 	if err = q.client.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: "quilkin-" + value}, cm); err != nil {
			// 		q.logger.Warnw("Error getting configmap", "namespace", pod.Namespace, "name", "quilkin-"+value, "error", err.Error())
			// 	}
			// 	if err := q.client.Delete(ctx, cm); err != nil {
			// 		q.logger.Warnw("Error deleting configmap", "namespace", pod.Namespace, "name", "quilkin-"+value, "error", err.Error())
			// 	}
			// }
		}

		controllerutil.RemoveFinalizer(pod, Finalizer)
		q.logger.Infow("Removing quilkin finalizer", "pod", pod.Name)
		if err := q.client.Update(ctx, pod); err != nil {
			q.logger.Warnw("failure reconciling. Requeuing pod.", "error", err.Error())
			return reconcile.Result{
				Requeue: true,
			}, nil
		}
	} else if pod.Status.Phase == corev1.PodRunning && pod.DeletionTimestamp.IsZero() {
		if isReceiver(pod) {
			q.handleRunningReceiver(pod)
		} else if isSender(pod) {
			q.handleRunningSender(pod)
		}
	}
	return reconcile.Result{}, nil
}

// handleRunningReceiver This adds the receiver to the xds node
// This function assumes the pod has already had its annotations checked for the correct one
func (q *QuilkinReconciler) handleRunningReceiver(pod *corev1.Pod) {
	value := pod.Annotations[ReceiverAnnotation]
	proxyName, port, err := parseReceiveAnnotation(value)
	if err != nil {
		q.logger.Errorw("Error parsing annotation", "annotation", value)
	}
	q.logger.Infow("Adding receiver", "proxy", proxyName, "port", port, "pod", pod.Status.PodIP)
	q.store.AddReceiver(proxyName, port, pod.Status.PodIP, pod.Name)
}

// handleRunningReceiver This adds the sender to the internal store
// This function assumes the pod has already had its annotations checked for the correct one
func (q *QuilkinReconciler) handleRunningSender(pod *corev1.Pod) {
	value := pod.Annotations[SenderAnnotation]
	q.logger.Infow("Adding sender", "proxy", value)
	q.store.AddSender(value, pod.Name)
}

// parseReceiveAnnotation validates and parses the string provided and returns the proxyName and port
// if they are valid.
func parseReceiveAnnotation(annotation string) (string, int, error) {
	annotationValues := strings.Split(annotation, ":")
	if len(annotationValues) != 2 {
		return "", 0, errors.New("annotation is not valid proxyname:port Combo")
	}
	proxyName := annotationValues[0]
	port, err := net.ParsePort(annotationValues[1], false)
	if err != nil {
		return "", 0, errors.New("annotation port is not a valid port")
	}
	return proxyName, port, nil
}

// isReceiver returns whether the pod is a receiver or not
func isReceiver(pod *corev1.Pod) bool {
	_, ok := pod.Annotations[ReceiverAnnotation]
	return ok
}

// isReceiver returns whether the pod is a receiver or not
func isSender(pod *corev1.Pod) bool {
	_, ok := pod.Annotations[SenderAnnotation]
	return ok
}

// containsString returns whether or not the slice provided contains the string provided.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
