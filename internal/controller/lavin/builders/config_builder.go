package builder

import (
	"fmt"
	"reflect"
	"strings"

	ini "gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceConfigBuilder struct {
	*ResourceBuilder
}

var (
	defaultConfig = `
	[main]
log_level = info
data_dir = /var/lib/lavinmq

[mgmt]
bind = 0.0.0.0

[amqp]
bind = 0.0.0.0
heartbeat = 300
	`

	clusteringConfig = `
[clustering]
enabled = true
bind = 0.0.0.0
port = 5679
`
)

func (builder *ResourceBuilder) ConfigBuilder() *ServiceConfigBuilder {
	return &ServiceConfigBuilder{
		ResourceBuilder: builder,
	}
}

func (b *ServiceConfigBuilder) Name() string {
	return "config"
}

func (b *ServiceConfigBuilder) NewObject() client.Object {
	labels := map[string]string{
		"app.kubernetes.io/name":       "lavinmq",
		"app.kubernetes.io/managed-by": "lavinmq-operator",
		"app.kubernetes.io/instance":   b.Instance.Name,
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", b.Instance.Name),
			Namespace: b.Instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{},
	}
}

// BuildConfigMap creates a ConfigMap for LavinMQ configuration
func (b *ServiceConfigBuilder) Build() (client.Object, error) {

	configMap := b.NewObject().(*corev1.ConfigMap)

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
		if port.Name == "https" {
			defaultConfig.Section("mgmt").Key("tls_port").SetValue(fmt.Sprintf("%d", port.ContainerPort))
		}
		if port.Name == "amqps" {
			defaultConfig.Section("amqp").Key("tls_port").SetValue(fmt.Sprintf("%d", port.ContainerPort))
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

	if b.Instance.Spec.Secrets != nil {
		defaultConfig.Section("main").Key("tls_cert").SetValue(fmt.Sprintf("/etc/lavinmq/tls/%s", "tls.crt"))
		defaultConfig.Section("main").Key("tls_key").SetValue(fmt.Sprintf("/etc/lavinmq/tls/%s", "tls.key"))
	}

	_, err = defaultConfig.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	_, err = clusterConfig.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write cluster config: %w", err)
	}

	configMap.Data["lavinmq.ini"] = config.String()

	return configMap, nil
}

func (b *ServiceConfigBuilder) Diff(old, new client.Object) (client.Object, bool, error) {
	oldConfigMap := old.(*corev1.ConfigMap)
	newConfigMap := new.(*corev1.ConfigMap)
	changed := false
	if !reflect.DeepEqual(oldConfigMap.Data["lavinmq.ini"], newConfigMap.Data["lavinmq.ini"]) {
		oldConfigMap.Data["lavinmq.ini"] = newConfigMap.Data["lavinmq.ini"]
		changed = true
	}

	return oldConfigMap, changed, nil
}
