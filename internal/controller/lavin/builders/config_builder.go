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
log_level = info
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

	iniFile, err := ini.Load([]byte(defaultConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	for _, port := range b.Instance.Spec.Ports {
		if port.Name == "http" {
			iniFile.Section("mgmt").Key("port").SetValue(fmt.Sprintf("%d", port.ContainerPort))
		}
		if port.Name == "amqp" {
			iniFile.Section("amqp").Key("port").SetValue(fmt.Sprintf("%d", port.ContainerPort))
		}
	}

	config := strings.Builder{}

	_, err = iniFile.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	configMap.Data["lavinmq.ini"] = config.String()

	// Set owner reference
	if err := ctrl.SetControllerReference(b.Instance, configMap, b.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	return configMap, nil
}
