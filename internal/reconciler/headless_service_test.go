package reconciler_test

import (
	"lavinmq-operator/internal/reconciler"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestDefaultHeadlessService(t *testing.T) {
	instance := defaultInstance.DeepCopy()
	defer k8sClient.Delete(t.Context(), instance)

	rc := &reconciler.HeadlessServiceReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	rc.Reconcile(t.Context())

	service := &corev1.Service{}
	assert.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service))
	assert.Equal(t, service.Name, instance.Name)
	assert.Equal(t, service.Spec.ClusterIP, "None")
	assert.Len(t, service.Spec.Ports, 3)
}

func TestCustomPorts(t *testing.T) {
	instance := defaultInstance.DeepCopy()
	defer k8sClient.Delete(t.Context(), instance)

	instance.Spec.Config.Amqp.Port = 1111
	instance.Spec.Config.Mgmt.Port = 2222
	instance.Spec.Config.Amqp.TlsPort = 3333
	instance.Spec.Config.Mgmt.TlsPort = 4444
	instance.Spec.Config.Mqtt.Port = 5555

	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	rc := &reconciler.HeadlessServiceReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	rc.Reconcile(t.Context())

	service := &corev1.Service{}
	assert.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service))
	assert.Len(t, service.Spec.Ports, 5)

	i := slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "amqp"
	})
	assert.Equal(t, service.Spec.Ports[i].Port, int32(1111))
	i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "http"
	})
	assert.NotEqual(t, i, -1)
	assert.Equal(t, service.Spec.Ports[i].Port, int32(2222))
	i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "amqps"
	})
	assert.NotEqual(t, i, -1)
	assert.Equal(t, service.Spec.Ports[i].Port, int32(3333))
	i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "https"
	})
	assert.NotEqual(t, i, -1)
	assert.Equal(t, service.Spec.Ports[i].Port, int32(4444))
	i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "mqtt"
	})
	assert.NotEqual(t, i, -1)
	assert.Equal(t, service.Spec.Ports[i].Port, int32(5555))
}

func TestClusteringPort(t *testing.T) {
	instance := defaultInstance.DeepCopy()
	defer k8sClient.Delete(t.Context(), instance)

	instance.Spec.EtcdEndpoints = []string{"etcd-0:2379"}
	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	rc := &reconciler.HeadlessServiceReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	rc.Reconcile(t.Context())

	service := &corev1.Service{}
	assert.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service))
	assert.Len(t, service.Spec.Ports, 4)
	idx := slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "clustering"
	})
	assert.NotEqual(t, idx, -1)
	assert.Equal(t, service.Spec.Ports[idx].Port, int32(5679))
}

func TestPortChanges(t *testing.T) {
	instance := defaultInstance.DeepCopy()
	defer k8sClient.Delete(t.Context(), instance)

	instance.Spec.Config.Amqp.Port = 5672
	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	rc := &reconciler.HeadlessServiceReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	rc.Reconcile(t.Context())

	service := &corev1.Service{}
	assert.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service))
	idx := slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "amqp"
	})
	assert.Equal(t, service.Spec.Ports[idx].Port, int32(5672))

	instance.Spec.Config.Amqp.Port = 1111
	assert.NoError(t, k8sClient.Update(t.Context(), instance))

	rc.Reconcile(t.Context())

	assert.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service))
	idx = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "amqp"
	})
	assert.Equal(t, service.Spec.Ports[idx].Port, int32(1111))
}
