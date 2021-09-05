package pod

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nfowl/quilkin-controller/pkg/store"
	"go.uber.org/zap"
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
	logger  *zap.SugaredLogger
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

	value, ok := pod.Annotations[ReceiverAnnotation]
	if ok {
		annotationValues := strings.Split(value, ":")
		if len(annotationValues) != 2 {
			q.logger.Errorw("Annotation is not valid proxyname:port Combo", "value", value)
		}
		proxyName := annotationValues[0]
		port, err := net.ParsePort(annotationValues[1], false)
		if err != nil {
			q.logger.Errorw("Annotation port is not a valid port", "port", annotationValues[1])
		}
		store.AddReceiver(proxyName, port, pod.Status.PodIP, pod.Name)
	}

	value, ok = pod.Annotations[SenderAnnotation]
	if ok {
		store.AddSender(value)
		//TODO ADD POD INJECTION HERE WITH QUILKIN CONFIG
		ns := pod.Namespace
		cm := &v1.ConfigMap{}
		err := q.Client.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: "quilkin-" + value}, cm)
		if err != nil {
			//Create new cm
			cm.Name = "quilkin-" + value
			cm.Namespace = pod.Namespace
			cm.Labels["managed-by"] = "quilkin-controller"
		}
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
