package builder

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	ini "gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ConfigReconciler struct {
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

func (builder *ResourceBuilder) ConfigReconciler() *ConfigReconciler {
	return &ConfigReconciler{
		ResourceBuilder: builder,
	}
}

func (b *ConfigReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	configMap, err := b.newObject()
	if err != nil {
		return ctrl.Result{}, err
	}

	err = b.GetItem(ctx, configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			b.CreateItem(ctx, configMap)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	err = b.updateFields(ctx, configMap)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = b.Client.Update(ctx, configMap)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (b *ConfigReconciler) newObject() (*corev1.ConfigMap, error) {
	labels := map[string]string{
		"app.kubernetes.io/name":       "lavinmq",
		"app.kubernetes.io/managed-by": "lavinmq-operator",
		"app.kubernetes.io/instance":   b.Instance.Name,
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.Instance.Name,
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

	if b.Instance.Spec.Config.ConsumerTimeout != 0 {
		defaultConfig.Section("main").Key("consumer_timeout").SetValue(fmt.Sprintf("%d", b.Instance.Spec.Config.ConsumerTimeout))
	}

	// TODO: Add advertised uri may be wrong here. Headless service?
	clusterConfig.Section("clustering").Key("advertised_uri").SetValue(fmt.Sprintf("tcp://%s:5679", b.Instance.Name))

	config := strings.Builder{}

	if b.Instance.Spec.TlsSecret != nil {
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

func (b *ConfigReconciler) updateFields(ctx context.Context, configMap *corev1.ConfigMap) error {
	newConfigMap, err := b.newObject()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(configMap.Data["lavinmq.ini"], newConfigMap.Data["lavinmq.ini"]) {
		configMap.Data["lavinmq.ini"] = newConfigMap.Data["lavinmq.ini"]
	}

	return nil
}

// Name returns the name of the config reconciler
func (b *ConfigReconciler) Name() string {
	return "config"
}
