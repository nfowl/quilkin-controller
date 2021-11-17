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
	"encoding/json"
	"errors"
	"net/http"

	"github.com/nfowl/quilkin-controller/internal/quilkin"
	"github.com/nfowl/quilkin-controller/internal/store"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	// Annotation key used to indicate a pod is receiver udp traffic from a quilkin sender
	ReceiverAnnotation = "nfowler.dev/quilkin.receiver"
	// Annotation key used to indicate a pod is sending udp traffic to any listening receivers
	SenderAnnotation = "nfowler.dev/quilkin.sender"
	// The finalizer string used to cleanup and setup senders/receivers as part of the reconcile action
	Finalizer = "quilkin.nfowler.dev/finalizer"
)

var (
	// The image source that will be injected in as a sidecar to senders
	QuilkinImage = "us-docker.pkg.dev/quilkin/release/quilkin:0.1.0"
)

type QuilkinAnnotationReader struct {
	client  client.Client
	decoder *admission.Decoder
	logger  *zap.SugaredLogger
	store   *store.SotwStore
}

func NewQuilkinAnnotationReader(c client.Client, l *zap.SugaredLogger, s *store.SotwStore) *QuilkinAnnotationReader {
	return &QuilkinAnnotationReader{
		client: c,
		logger: l,
		store:  s,
	}
}

// InjectDecoder injects the admission decoder into the QuilkinAnnotationReader provided
func (q *QuilkinAnnotationReader) InjectDecoder(d *admission.Decoder) error {
	q.decoder = d
	return nil
}

// Handle is the function that handles all webhook admission requests
// Currently this only acts on Create requests. Updates/Deletes are mostly handled as part of the reconciler.
func (q *QuilkinAnnotationReader) Handle(ctx context.Context, req admission.Request) admission.Response {
	//Handle updates/creates
	if *req.DryRun {
		return admission.Allowed("Dry Run Ignored")
	}

	if req.Operation == admissionv1.Delete {
		return admission.Allowed("NO OP")
	}
	if req.Operation == admissionv1.Create {
		return q.handleCreate(ctx, req)
	}
	if req.Operation == admissionv1.Update {
		return admission.Allowed("NO OP")
	}

	return admission.Errored(http.StatusInternalServerError, errors.New("failed to run webhook"))
}

// handleCreate handles new pods by adding the finalizer and if the pod is a sender it will inject
// the sidecar proxy
func (q *QuilkinAnnotationReader) handleCreate(ctx context.Context, req admission.Request) admission.Response {
	pod := &v1.Pod{}
	err := q.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if !HasAnnotations(pod) || !pod.DeletionTimestamp.IsZero() {
		return admission.Allowed("No changes required")
	}

	_, ok := pod.Annotations[ReceiverAnnotation]
	if ok {
		q.logger.Infow("Adding receiver finalizer")
		controllerutil.AddFinalizer(pod, Finalizer)
	}

	value, ok2 := pod.Annotations[SenderAnnotation]
	if ok2 {
		q.logger.Infow("Adding sender", "pod", pod.Name)
		cm := &v1.ConfigMap{}
		err := q.client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: "quilkin-" + value}, cm)
		if err != nil {
			conf, err := yaml.Marshal(quilkin.NewQuilkinConfig(value))
			if err != nil {
				q.logger.Errorw("Error building Quilkin config", "error", err.Error())
			}
			//Create new cm
			cm.Name = "quilkin-" + value
			cm.Namespace = req.Namespace
			cm.Labels = make(map[string]string)
			cm.Labels["managed-by"] = "quilkin-controller"
			cm.Data = make(map[string]string)
			cm.Data["quilkin.yaml"] = string(conf)
			err = q.client.Create(ctx, cm)
			if err != nil {
				q.logger.Errorw("Error Creating Configmap", "error", err.Error())
			}
		}
		container := makeQuilkinContainer()
		q.logger.Infow("Adding sender finalizer", "pod", pod.Name)
		controllerutil.AddFinalizer(pod, Finalizer)
		pod.Spec.Containers = append(pod.Spec.Containers, container)
		pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{Name: "quilkin-config", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: "quilkin-" + value}}}})
	}
	marshaledPod, err := json.Marshal(pod)

	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// makeQuilkinContainer constructs the sidecar container definition
func makeQuilkinContainer() v1.Container {
	volumes := make([]v1.VolumeMount, 0, 1)
	volumes = append(volumes, v1.VolumeMount{Name: "quilkin-config", ReadOnly: true, MountPath: "/etc/quilkin"})
	ports := make([]v1.ContainerPort, 0, 1)
	ports = append(ports, v1.ContainerPort{Name: "http-admin", ContainerPort: 9091, Protocol: v1.ProtocolTCP})
	return v1.Container{
		Name:         "quilkin",
		Image:        QuilkinImage,
		VolumeMounts: volumes,
		Ports:        ports,
	}
}
