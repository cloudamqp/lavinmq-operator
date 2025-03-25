package builder

import (
	"fmt"
	"strings"

	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"

	ini "gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ServiceConfigBuilder struct {
	Instance *cloudamqpcomv1alpha1.LavinMQ
	Scheme   *runtime.Scheme
}

var (
	defaultConfig = `
	[main]
log_level = debug
data_dir = /var/lib/lavinmq

[mgmt]
bind = 0.0.0.0
;unix_path = /run/lavinmq/http.sock

[amqp]
bind = 0.0.0.0
heartbeat = 300
;unix_path = /run/lavinmq/amqp.sock
;unix_proxy_protocol = 1
	`

	clusteringConfig = `
[clustering]
enabled = true
bind = 0.0.0.0
port = 5679
`
)

// BuildConfigMap creates a ConfigMap for LavinMQ configuration
func (b *ServiceConfigBuilder) Build() (*corev1.ConfigMap, error) {
	labels := map[string]string{
		"app.kubernetes.io/name":       "lavinmq",
		"app.kubernetes.io/managed-by": "lavinmq-operator",
		"app.kubernetes.io/instance":   b.Instance.Name,
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", b.Instance.Name),
			Namespace: b.Instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{},
	}

	defaultConfig, err := ini.Load([]byte(defaultConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	for _, port := range b.Instance.Spec.Ports {
		if port.Name == "http" {
			defaultConfig.Section("mgmt").Key("port").SetValue(fmt.Sprintf("%d", port.ContainerPort))
		}
		if port.Name == "amqp" {
			defaultConfig.Section("amqp").Key("port").SetValue(fmt.Sprintf("%d", port.ContainerPort))
		}
	}

	clusterConfig, err := ini.Load([]byte(clusteringConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster config: %w", err)
	}

	if b.Instance.Spec.EtcdEndpoints != nil {
		clusterConfig.Section("clustering").Key("etcd_endpoints").SetValue(strings.Join(b.Instance.Spec.EtcdEndpoints, ","))
	}

	// TODO: Add advertised uri may be wrong here. Headless service?
	clusterConfig.Section("clustering").Key("advertised_uri").SetValue(fmt.Sprintf("tcp://%s:5679", b.Instance.Name))

	config := strings.Builder{}

	_, err = defaultConfig.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	_, err = clusterConfig.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write cluster config: %w", err)
	}

	configMap.Data["lavinmq.ini"] = config.String()

	// Set owner reference
	if err := ctrl.SetControllerReference(b.Instance, configMap, b.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	return configMap, nil
}
