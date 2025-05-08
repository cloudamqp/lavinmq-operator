package reconciler_test

import (
	"context"
	"lavinmq-operator/internal/reconciler"
	testutils "lavinmq-operator/internal/test_utils"
	"testing"

	"github.com/stretchr/testify/assert"
	ini "gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func verifyConfigMapEquality(t *testing.T, configMap *corev1.ConfigMap, expectedConfig string) {
	conf, _ := ini.Load([]byte(configMap.Data[reconciler.ConfigFileName]))
	expectedConf, _ := ini.Load([]byte(expectedConfig))

	for _, section := range conf.Sections() {
		for _, key := range section.Keys() {
			val := conf.Section(section.Name()).Key(key.Name()).Value()
			assert.Equal(t, expectedConf.Section(section.Name()).Key(key.Name()).Value(), val)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

	defer k8sClient.Delete(t.Context(), instance)

	rc := &reconciler.ConfigReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	rc.Reconcile(t.Context())

	var expectedConfig = `
			[main]
			data_dir = /var/lib/lavinmq

			[mgmt]
			bind = 0.0.0.0
			port = 15672

			[amqp]
			bind = 0.0.0.0
			port = 5672

			[mqtt]
			bind = 0.0.0.0
			port = 1883

			[clustering]
			bind = 0.0.0.0
			port = 5679
	`
	rc.Reconcile(context.Background())

	configMap := &corev1.ConfigMap{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, configMap)
	assert.NoError(t, err)
	assert.Equal(t, instance.Name, configMap.Name)
	verifyConfigMapEquality(t, configMap, expectedConfig)
}

func TestCustomConfigPorts(t *testing.T) {
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

	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	expectedConfig := `
	[main]
	data_dir = /var/lib/lavinmq

	[mgmt]
	bind = 0.0.0.0
	port = 2222
	tls_port = 4444

	[amqp]
	bind = 0.0.0.0
	port = 1111
	tls_port = 3333

	[mqtt]
	bind = 0.0.0.0
	port = 1883

	[clustering]
	bind = 0.0.0.0
	port = 5679
`

	rc := &reconciler.ConfigReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	rc.Reconcile(t.Context())
	configMap := &corev1.ConfigMap{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, configMap)
	assert.NoError(t, err)
	assert.Equal(t, instance.Name, configMap.Name)
	verifyConfigMapEquality(t, configMap, expectedConfig)
}

func TestDisablingNonTlsPorts(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

	defer k8sClient.Delete(t.Context(), instance)

	instance.Spec.Config.Amqp.Port = -1
	instance.Spec.Config.Mgmt.Port = -1
	instance.Spec.Config.Mqtt.Port = -1
	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	expectedConfig := `
	[main]
	data_dir = /var/lib/lavinmq

	[mgmt]
	bind = 0.0.0.0
	port = -1

	[amqp]
	bind = 0.0.0.0
	port = -1

	[mqtt]
	bind = 0.0.0.0
	port = -1

	[clustering]
	bind = 0.0.0.0
	port = 5679
`

	rc := &reconciler.ConfigReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	rc.Reconcile(t.Context())

	configMap := &corev1.ConfigMap{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, configMap)
	assert.NoError(t, err)
	assert.Equal(t, instance.Name, configMap.Name)
	verifyConfigMapEquality(t, configMap, expectedConfig)
}
