package reconciler_test

import (
	"context"
	"testing"

	v1alpha1 "github.com/cloudamqp/lavinmq-operator/api/v1alpha1"
	"github.com/cloudamqp/lavinmq-operator/internal/reconciler"
	testutils "github.com/cloudamqp/lavinmq-operator/internal/test_utils"

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

func TestSniConfig(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)
	defer k8sClient.Delete(t.Context(), instance)

	instance.Spec.Config.Sni = []v1alpha1.SniConfig{
		{
			Hostname: "example.com",
			TlsSecret: corev1.SecretReference{
				Name: "example-tls",
			},
		},
	}

	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	expectedConfig := `
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

	[sni:example.com]
	tls_cert = /etc/lavinmq/sni/example-com/tls.crt
	tls_key = /etc/lavinmq/sni/example-com/tls.key
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
	verifyConfigMapEquality(t, configMap, expectedConfig)
}

func TestSniConfigWithMtls(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)
	defer k8sClient.Delete(t.Context(), instance)

	instance.Spec.Config.Sni = []v1alpha1.SniConfig{
		{
			Hostname: "secure.example.com",
			TlsSecret: corev1.SecretReference{
				Name: "secure-tls",
			},
			TlsCaSecret: &corev1.SecretReference{
				Name: "secure-ca",
			},
			TlsVerifyPeer: true,
		},
	}

	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	expectedConfig := `
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

	[sni:secure.example.com]
	tls_cert = /etc/lavinmq/sni/secure-example-com/tls.crt
	tls_key = /etc/lavinmq/sni/secure-example-com/tls.key
	tls_ca_cert = /etc/lavinmq/sni/secure-example-com-ca/ca.crt
	tls_verify_peer = true
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
	verifyConfigMapEquality(t, configMap, expectedConfig)
}

func TestSniConfigWithProtocolOverrides(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)
	defer k8sClient.Delete(t.Context(), instance)

	verifyPeer := true
	instance.Spec.Config.Sni = []v1alpha1.SniConfig{
		{
			Hostname: "multi-protocol.example.com",
			TlsSecret: corev1.SecretReference{
				Name: "default-tls",
			},
			Amqp: &v1alpha1.SniProtocolConfig{
				TlsSecret: &corev1.SecretReference{
					Name: "amqp-tls",
				},
				TlsVerifyPeer: &verifyPeer,
			},
			Mqtt: &v1alpha1.SniProtocolConfig{
				TlsSecret: &corev1.SecretReference{
					Name: "mqtt-tls",
				},
			},
			Http: &v1alpha1.SniProtocolConfig{
				TlsSecret: &corev1.SecretReference{
					Name: "http-tls",
				},
			},
		},
	}

	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	expectedConfig := `
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

	[sni:multi-protocol.example.com]
	tls_cert = /etc/lavinmq/sni/multi-protocol-example-com/tls.crt
	tls_key = /etc/lavinmq/sni/multi-protocol-example-com/tls.key
	amqp_tls_cert = /etc/lavinmq/sni/multi-protocol-example-com-amqp/tls.crt
	amqp_tls_key = /etc/lavinmq/sni/multi-protocol-example-com-amqp/tls.key
	amqp_tls_verify_peer = true
	mqtt_tls_cert = /etc/lavinmq/sni/multi-protocol-example-com-mqtt/tls.crt
	mqtt_tls_key = /etc/lavinmq/sni/multi-protocol-example-com-mqtt/tls.key
	http_tls_cert = /etc/lavinmq/sni/multi-protocol-example-com-http/tls.crt
	http_tls_key = /etc/lavinmq/sni/multi-protocol-example-com-http/tls.key
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
	verifyConfigMapEquality(t, configMap, expectedConfig)
}

func TestMultipleSniHosts(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)
	defer k8sClient.Delete(t.Context(), instance)

	instance.Spec.Config.Sni = []v1alpha1.SniConfig{
		{
			Hostname: "tenant1.example.com",
			TlsSecret: corev1.SecretReference{
				Name: "tenant1-tls",
			},
		},
		{
			Hostname: "tenant2.example.com",
			TlsSecret: corev1.SecretReference{
				Name: "tenant2-tls",
			},
		},
		{
			Hostname: "*.wildcard.com",
			TlsSecret: corev1.SecretReference{
				Name: "wildcard-tls",
			},
		},
	}

	assert.NoError(t, k8sClient.Create(t.Context(), instance))

	expectedConfig := `
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

	[sni:tenant1.example.com]
	tls_cert = /etc/lavinmq/sni/tenant1-example-com/tls.crt
	tls_key = /etc/lavinmq/sni/tenant1-example-com/tls.key

	[sni:tenant2.example.com]
	tls_cert = /etc/lavinmq/sni/tenant2-example-com/tls.crt
	tls_key = /etc/lavinmq/sni/tenant2-example-com/tls.key

	[sni:*.wildcard.com]
	tls_cert = /etc/lavinmq/sni/wildcard-wildcard-com/tls.crt
	tls_key = /etc/lavinmq/sni/wildcard-wildcard-com/tls.key
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
	verifyConfigMapEquality(t, configMap, expectedConfig)
}
