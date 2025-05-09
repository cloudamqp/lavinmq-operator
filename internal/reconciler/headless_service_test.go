package reconciler_test

import (
	"slices"
	"testing"

	"github.com/cloudamqp/lavinmq-operator/internal/reconciler"
	testutils "github.com/cloudamqp/lavinmq-operator/internal/test_utils"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestDefaultHeadlessService(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

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
	assert.Equal(t, instance.Name, service.Name)
	assert.Equal(t, "None", service.Spec.ClusterIP)
	assert.Len(t, service.Spec.Ports, 3)
}

func TestCustomPorts(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

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
	assert.Equal(t, int32(1111), service.Spec.Ports[i].Port)
	i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "http"
	})
	assert.NotEqual(t, i, -1)
	assert.Equal(t, int32(2222), service.Spec.Ports[i].Port)
	i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "amqps"
	})
	assert.NotEqual(t, i, -1)
	assert.Equal(t, int32(3333), service.Spec.Ports[i].Port)
	i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "https"
	})
	assert.NotEqual(t, i, -1)
	assert.Equal(t, int32(4444), service.Spec.Ports[i].Port)
	i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "mqtt"
	})
	assert.NotEqual(t, i, -1)
	assert.Equal(t, int32(5555), service.Spec.Ports[i].Port)
}

func TestClusteringPort(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

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
	assert.Equal(t, int32(5679), service.Spec.Ports[idx].Port)
}

func TestPortChanges(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

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
	assert.Equal(t, int32(5672), service.Spec.Ports[idx].Port)

	instance.Spec.Config.Amqp.Port = 1111
	assert.NoError(t, k8sClient.Update(t.Context(), instance))

	rc.Reconcile(t.Context())

	assert.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service))
	idx = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
		return port.Name == "amqp"
	})
	assert.Equal(t, int32(1111), service.Spec.Ports[idx].Port)
}
