package reconciler

import (
	"context"
	"lavinmq-operator/internal/controller/utils"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type HeadlessServiceReconciler struct {
	*ResourceReconciler
}

func (reconciler *ResourceReconciler) HeadlessServiceReconciler() *HeadlessServiceReconciler {
	return &HeadlessServiceReconciler{
		ResourceReconciler: reconciler,
	}
}

func (b *HeadlessServiceReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	service := b.newObject()

	err := b.GetItem(ctx, service)
	if err != nil {
		if apierrors.IsNotFound(err) {
			b.CreateItem(ctx, service)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	b.updateFields(ctx, service)

	err = b.Client.Update(ctx, service)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (b *HeadlessServiceReconciler) newObject() *corev1.Service {
	servicePorts := []corev1.ServicePort{}
	if b.Instance.Spec.EtcdEndpoints != nil {
		servicePorts = appendServicePorts(servicePorts, 5679, "clustering")
	}

	if b.Instance.Spec.Config.Mgmt.Port == 0 {
		servicePorts = appendServicePorts(servicePorts, 15672, "http")
	} else {
		servicePorts = appendServicePorts(servicePorts, b.Instance.Spec.Config.Mgmt.Port, "http")
	}

	if b.Instance.Spec.Config.Mgmt.TlsPort != 0 {
		servicePorts = appendServicePorts(servicePorts, b.Instance.Spec.Config.Mgmt.TlsPort, "https")
	}

	if b.Instance.Spec.Config.Amqp.Port == 0 {
		servicePorts = appendServicePorts(servicePorts, 5672, "amqp")
	} else {
		servicePorts = appendServicePorts(servicePorts, b.Instance.Spec.Config.Amqp.Port, "amqp")
	}

	if b.Instance.Spec.Config.Amqp.TlsPort != 0 {
		servicePorts = appendServicePorts(servicePorts, b.Instance.Spec.Config.Amqp.TlsPort, "amqps")
	}

	if b.Instance.Spec.Config.Mqtt.Port == 0 {
		servicePorts = appendServicePorts(servicePorts, 1883, "mqtt")
	} else {
		servicePorts = appendServicePorts(servicePorts, b.Instance.Spec.Config.Mqtt.Port, "mqtt")
	}

	if b.Instance.Spec.Config.Mqtt.TlsPort != 0 {
		servicePorts = appendServicePorts(servicePorts, b.Instance.Spec.Config.Mqtt.TlsPort, "mqtts")
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.Instance.Name,
			Namespace: b.Instance.Namespace,
			Labels:    utils.LabelsForLavinMQ(b.Instance),
		},
		Spec: corev1.ServiceSpec{
			Selector:  b.Instance.Labels,
			ClusterIP: "None",
			Ports:     servicePorts,
		},
	}

	return service
}

func appendServicePorts(servicePorts []corev1.ServicePort, port int32, name string) []corev1.ServicePort {
	servicePorts = append(servicePorts, corev1.ServicePort{
		Name:       name,
		Port:       port,
		TargetPort: intstr.FromInt(int(port)),
		Protocol:   "TCP",
	})
	return servicePorts
}

func (b *HeadlessServiceReconciler) updateFields(ctx context.Context, service *corev1.Service) {
	newService := b.newObject()

	if !reflect.DeepEqual(service.Spec.Ports, newService.Spec.Ports) {
		service.Spec.Ports = newService.Spec.Ports
	}
}

// Name returns the name of the headless service reconciler
func (b *HeadlessServiceReconciler) Name() string {
	return "headless-service"
}
