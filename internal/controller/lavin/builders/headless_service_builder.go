package builder

import (
	"fmt"
	"lavinmq-operator/internal/controller/utils"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HeadlessServiceBuilder struct {
	*ResourceBuilder
}

func (builder *ResourceBuilder) HeadlessServiceBuilder() *HeadlessServiceBuilder {
	return &HeadlessServiceBuilder{
		ResourceBuilder: builder,
	}
}

func (b *HeadlessServiceBuilder) Name() string {
	return "service"
}

func (b *HeadlessServiceBuilder) NewObject() client.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-service", b.Instance.Name),
			Namespace: b.Instance.Namespace,
			Labels:    utils.LabelsForLavinMQ(b.Instance),
		},
	}
}

func (b *HeadlessServiceBuilder) Build() (client.Object, error) {
	service := b.NewObject().(*corev1.Service)
	servicePorts := []corev1.ServicePort{}
	if b.Instance.Spec.EtcdEndpoints != nil {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       "clustering",
			Port:       5679,
			TargetPort: intstr.FromInt(5679),
			Protocol:   "TCP",
		})
	}

	for _, port := range b.Instance.Spec.Ports {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       port.Name,
			Port:       port.ContainerPort,
			TargetPort: intstr.FromInt(int(port.ContainerPort)),
			Protocol:   "TCP",
		})
	}

	service.Spec = corev1.ServiceSpec{
		Selector:  b.Instance.Labels,
		ClusterIP: "None",
		Ports:     servicePorts,
	}

	return service, nil
}

func (b *HeadlessServiceBuilder) Diff(oldObj, newObj client.Object) (client.Object, bool, error) {
	oldService := oldObj.(*corev1.Service)
	newService := newObj.(*corev1.Service)
	changed := false

	if !reflect.DeepEqual(oldService.Spec.Ports, newService.Spec.Ports) {
		oldService.Spec.Ports = newService.Spec.Ports
		changed = true
	}

	return oldService, changed, nil
}
