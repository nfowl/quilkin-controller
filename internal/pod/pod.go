package pod

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nfowl/quilkin-controller/internal/quilkin"
	"github.com/nfowl/quilkin-controller/internal/store"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	ReceiverAnnotation = "nfowler.dev/quilkin.receiver"
	SenderAnnotation   = "nfowler.dev/quilkin.sender"
)

type QuilkinAnnotationReader struct {
	Client  client.Client
	decoder *admission.Decoder
	Logger  *zap.SugaredLogger
}

func (q *QuilkinAnnotationReader) InjectDecoder(d *admission.Decoder) error {
	q.decoder = d
	return nil
}

func (q *QuilkinAnnotationReader) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := q.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if *req.DryRun {
		return admission.Allowed("Dry Run Ignored")
	}

	if req.Operation == admissionv1.Create {
		pod = q.HandleCreate(ctx, pod)
	}
	//TODO HANDLE DELETE AND UPDATE

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

///HandleCreate handles new pods by injecting the sidecar for senders or adding it to the
///xds node list for receivers
func (q *QuilkinAnnotationReader) HandleCreate(ctx context.Context, pod *v1.Pod) *v1.Pod {
	value, ok := pod.Annotations[ReceiverAnnotation]
	if ok {
		annotationValues := strings.Split(value, ":")
		if len(annotationValues) != 2 {
			q.Logger.Errorw("Annotation is not valid proxyname:port Combo", "value", value)
		}
		proxyName := annotationValues[0]
		port, err := net.ParsePort(annotationValues[1], false)
		if err != nil {
			q.Logger.Errorw("Annotation port is not a valid port", "port", annotationValues[1])
		}
		q.Logger.Infow("Adding receiver", "proxy", proxyName, "port", port, "pod", pod.Name)
		store.AddReceiver(proxyName, port, pod.Status.PodIP, pod.Name)
	}

	value, ok = pod.Annotations[SenderAnnotation]
	if ok {
		store.AddSender(value)
		cm := &v1.ConfigMap{}
		err := q.Client.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: "quilkin-" + value}, cm)
		if err != nil {
			conf, err := yaml.Marshal(quilkin.NewQuilkinConfig(value))
			if err != nil {
				q.Logger.Errorw("Error building Quilkin config", "error", err.Error())
			}
			//Create new cm
			cm.Name = "quilkin-" + value
			cm.Namespace = pod.Namespace
			cm.Labels["managed-by"] = "quilkin-controller"
			cm.Data["quilkin.yaml"] = string(conf)
			err = q.Client.Create(ctx, cm)
			if err != nil {
				q.Logger.Errorw("Error Creating Configmap", "error", err.Error())
			}
		}

		container := makeQuilkinContainer()
		pod.Spec.Containers = append(pod.Spec.Containers, container)
		pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{Name: "quilkin-config", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: "quilkin-" + value}}}})
	}
	return pod
}

func makeQuilkinContainer() v1.Container {
	volumes := make([]v1.VolumeMount, 1)
	volumes = append(volumes, v1.VolumeMount{Name: "quilkin-config", ReadOnly: true, MountPath: "/etc/quilkin"})
	return v1.Container{
		Name:         "quilkin",
		Image:        "us-docker.pkg.dev/quilkin/release/quilkin:0.1.0",
		VolumeMounts: volumes,
	}
}
