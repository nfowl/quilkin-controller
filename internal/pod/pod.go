package pod

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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ReceiverAnnotation = "nfowler.dev/quilkin.receiver"
	SenderAnnotation   = "nfowler.dev/quilkin.sender"
	Finalizer          = "quilkin.nfowler.dev/finalizer"
)

type QuilkinAnnotationReader struct {
	client  client.Client
	decoder *admission.Decoder
	logger  *zap.SugaredLogger
	store   *store.SoTWStore
}

func NewQuilkinAnnotationReader(c client.Client, l *zap.SugaredLogger, s *store.SoTWStore) *QuilkinAnnotationReader {
	return &QuilkinAnnotationReader{
		client: c,
		logger: l,
		store:  s,
	}
}

func (q *QuilkinAnnotationReader) InjectDecoder(d *admission.Decoder) error {
	q.decoder = d
	return nil
}

func (q *QuilkinAnnotationReader) Handle(ctx context.Context, req admission.Request) admission.Response {
	//Handle updates/creates
	if *req.DryRun {
		return admission.Allowed("Dry Run Ignored")
	}

	if req.Operation == admissionv1.Delete {
		return q.handleDelete(ctx, req)
	}
	if req.Operation == admissionv1.Create {
		return q.handleCreate(ctx, req)
	}
	if req.Operation == admissionv1.Update {
		return admission.Allowed("NO OP")
	}

	return admission.Errored(http.StatusInternalServerError, errors.New("Failed to run webhook"))
}

func (q *QuilkinAnnotationReader) handleDelete(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := q.decoder.DecodeRaw(req.OldObject, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	//No changes required as pod is not part of
	if !HasAnnotations(pod) {
		return admission.Allowed("No changes required")
	}

	return admission.Allowed("NO OP")
}

///HandleCreate handles new pods by injecting the sidecar for senders or adding it to the
///xds node list for receivers
func (q *QuilkinAnnotationReader) handleCreate(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := q.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if !HasAnnotations(pod) {
		return admission.Allowed("No changes required")
	}

	_, ok := pod.Annotations[ReceiverAnnotation]
	if ok {
		q.logger.Infow("Adding receiver finalizer")
		controllerutil.AddFinalizer(pod, Finalizer)
	}

	value, ok2 := pod.Annotations[SenderAnnotation]
	if ok2 {
		q.logger.Infow("Adding sender")
		q.store.AddSender(value)
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
		q.logger.Infow("Adding sender finalizer")
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

func makeQuilkinContainer() v1.Container {
	volumes := make([]v1.VolumeMount, 0, 1)
	volumes = append(volumes, v1.VolumeMount{Name: "quilkin-config", ReadOnly: true, MountPath: "/etc/quilkin"})
	ports := make([]v1.ContainerPort, 0, 1)
	ports = append(ports, v1.ContainerPort{Name: "http-admin", ContainerPort: 9091, Protocol: corev1.ProtocolTCP})
	return v1.Container{
		Name:         "quilkin",
		Image:        "us-docker.pkg.dev/quilkin/release/quilkin:0.1.0",
		VolumeMounts: volumes,
		Ports:        ports,
	}
}
