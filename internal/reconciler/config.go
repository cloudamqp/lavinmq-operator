package reconciler

import (
	"context"
	"fmt"
	"lavinmq-operator/internal/controller/utils"
	"reflect"
	"strings"

	ini "gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ConfigReconciler struct {
	*ResourceReconciler
}

var (
	defaultConfig = `
[main]
data_dir = /var/lib/lavinmq

[mgmt]
bind = 0.0.0.0

[amqp]
bind = 0.0.0.0

[mqtt]
bind = 0.0.0.0
	`

	clusteringConfig = `
[clustering]
enabled = true
bind = 0.0.0.0
port = 5679
`
)

func (reconciler *ResourceReconciler) ConfigReconciler() *ConfigReconciler {
	return &ConfigReconciler{
		ResourceReconciler: reconciler,
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
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.Instance.Name,
			Namespace: b.Instance.Namespace,
			Labels:    utils.LabelsForLavinMQ(b.Instance),
		},
		Data: map[string]string{},
	}
	config := strings.Builder{}

	mainConfig, err := b.GenerateMainConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate main config: %w", err)
	}
	_, err = mainConfig.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write main config: %w", err)
	}

	amqpConfig, err := b.GenerateAmqpConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate amqp config: %w", err)
	}
	_, err = amqpConfig.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write amqp config: %w", err)
	}

	mqttConfig, err := b.GenerateMqttConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate mqtt config: %w", err)
	}
	_, err = mqttConfig.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write mqtt config: %w", err)
	}

	mgmtConfig, err := b.GenerateMgmtConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate mgmt config: %w", err)
	}
	_, err = mgmtConfig.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write mgmt config: %w", err)
	}

	clusterConfig, err := b.GenerateClusteringConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate clustering config: %w", err)
	}
	_, err = clusterConfig.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write cluster config: %w", err)
	}

	configMap.Data["lavinmq.ini"] = config.String()
	return configMap, nil
}

func (b *ConfigReconciler) GenerateMainConfig() (*ini.File, error) {
	cfg, err := ini.Load([]byte(defaultConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	mainConfig := b.Instance.Spec.Config.Main

	if &mainConfig.ConsumerTimeout != nil {
		cfg.Section("main").Key("consumer_timeout").SetValue(fmt.Sprintf("%d", mainConfig.ConsumerTimeout))
	}
	if &mainConfig.DefaultConsumerPrefetch != nil {
		cfg.Section("main").Key("default_consumer_prefetch").SetValue(fmt.Sprintf("%d", mainConfig.DefaultConsumerPrefetch))
	}
	if &mainConfig.DefaultPassword != nil {
		cfg.Section("main").Key("default_password").SetValue(mainConfig.DefaultPassword)
	}
	if &mainConfig.DefaultUser != nil {
		cfg.Section("main").Key("default_user").SetValue(mainConfig.DefaultUser)
	}
	if &mainConfig.FreeDiskMin != nil {
		cfg.Section("main").Key("free_disk_min").SetValue(fmt.Sprintf("%d", mainConfig.FreeDiskMin))
	}
	if &mainConfig.FreeDiskWarn != nil {
		cfg.Section("main").Key("free_disk_warn").SetValue(fmt.Sprintf("%d", mainConfig.FreeDiskWarn))
	}
	if &mainConfig.LogExchange != nil {
		cfg.Section("main").Key("log_exchange").SetValue(fmt.Sprintf("%t", mainConfig.LogExchange))
	}
	if &mainConfig.LogLevel != nil {
		cfg.Section("main").Key("log_level").SetValue(mainConfig.LogLevel)
	}
	if &mainConfig.MaxDeletedDefinitions != nil {
		cfg.Section("main").Key("max_deleted_definitions").SetValue(fmt.Sprintf("%d", mainConfig.MaxDeletedDefinitions))
	}
	if &mainConfig.SegmentSize != nil {
		cfg.Section("main").Key("segment_size").SetValue(fmt.Sprintf("%d", mainConfig.SegmentSize))
	}
	if &mainConfig.SetTimestamp != nil {
		cfg.Section("main").Key("set_timestamp").SetValue(fmt.Sprintf("%t", mainConfig.SetTimestamp))
	}
	if &mainConfig.SocketBufferSize != nil {
		cfg.Section("main").Key("socket_buffer_size").SetValue(fmt.Sprintf("%d", mainConfig.SocketBufferSize))
	}
	if &mainConfig.StatsInterval != nil {
		cfg.Section("main").Key("stats_interval").SetValue(fmt.Sprintf("%d", mainConfig.StatsInterval))
	}
	if &mainConfig.StatsLogSize != nil {
		cfg.Section("main").Key("stats_log_size").SetValue(fmt.Sprintf("%d", mainConfig.StatsLogSize))
	}
	if &mainConfig.TcpKeepalive != nil {
		cfg.Section("main").Key("tcp_keepalive").SetValue(mainConfig.TcpKeepalive)
	}
	if &mainConfig.TcpNodelay != nil {
		cfg.Section("main").Key("tcp_nodelay").SetValue(fmt.Sprintf("%t", mainConfig.TcpNodelay))
	}
	if &mainConfig.TlsCiphers != nil {
		cfg.Section("main").Key("tls_ciphers").SetValue(mainConfig.TlsCiphers)
	}
	if &mainConfig.TlsMinVersion != nil {
		cfg.Section("main").Key("tls_min_version").SetValue(mainConfig.TlsMinVersion)
	}
	if b.Instance.Spec.TlsSecret != nil {
		cfg.Section("main").Key("tls_cert").SetValue(fmt.Sprintf("/etc/lavinmq/tls/%s", "tls.crt"))
		cfg.Section("main").Key("tls_key").SetValue(fmt.Sprintf("/etc/lavinmq/tls/%s", "tls.key"))
	}
	return cfg, nil
}

func (b *ConfigReconciler) GenerateClusteringConfig() (*ini.File, error) {
	cfg, err := ini.Load([]byte(clusteringConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	cfg.Section("clustering").Key("etcd_prefix").SetValue(b.Instance.Name)

	if b.Instance.Spec.EtcdEndpoints != nil {
		cfg.Section("clustering").Key("etcd_endpoints").SetValue(strings.Join(b.Instance.Spec.EtcdEndpoints, ","))
	}

	if &b.Instance.Spec.Config.Clustering.MaxUnsyncedActions != nil {
		cfg.Section("clustering").Key("max_unsynced_actions").SetValue(fmt.Sprintf("%d", b.Instance.Spec.Config.Clustering.MaxUnsyncedActions))
	}

	return cfg, nil
}

func (b *ConfigReconciler) GenerateAmqpConfig() (*ini.File, error) {
	cfg, err := ini.Load([]byte(defaultConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	amqpConfig := b.Instance.Spec.Config.Amqp

	if &amqpConfig.ChannelMax != nil {
		cfg.Section("amqp").Key("channel_max").SetValue(fmt.Sprintf("%d", amqpConfig.ChannelMax))
	}
	if &amqpConfig.FrameMax != nil {
		cfg.Section("amqp").Key("frame_max").SetValue(fmt.Sprintf("%d", amqpConfig.FrameMax))
	}
	if &amqpConfig.Heartbeat != nil {
		cfg.Section("amqp").Key("heartbeat").SetValue(fmt.Sprintf("%d", amqpConfig.Heartbeat))
	}
	if &amqpConfig.MaxMessageSize != nil {
		cfg.Section("amqp").Key("max_message_size").SetValue(fmt.Sprintf("%d", amqpConfig.MaxMessageSize))
	}
	if &amqpConfig.Port != nil {
		cfg.Section("amqp").Key("port").SetValue(fmt.Sprintf("%d", amqpConfig.Port))
	}
	if &amqpConfig.TlsPort != nil {
		cfg.Section("amqp").Key("tls_port").SetValue(fmt.Sprintf("%d", amqpConfig.TlsPort))
	}

	return cfg, nil
}

func (b *ConfigReconciler) GenerateMqttConfig() (*ini.File, error) {
	cfg, err := ini.Load([]byte(defaultConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	mqttConfig := b.Instance.Spec.Config.Mqtt

	if &mqttConfig.MaxInflightMessages != nil {
		cfg.Section("mqtt").Key("max_inflight_messages").SetValue(fmt.Sprintf("%d", mqttConfig.MaxInflightMessages))
	}
	if &mqttConfig.Port != nil {
		cfg.Section("mqtt").Key("port").SetValue(fmt.Sprintf("%d", mqttConfig.Port))
	}
	if &mqttConfig.TlsPort != nil {
		cfg.Section("mqtt").Key("tls_port").SetValue(fmt.Sprintf("%d", mqttConfig.TlsPort))
	}

	return cfg, nil
}

func (b *ConfigReconciler) GenerateMgmtConfig() (*ini.File, error) {
	cfg, err := ini.Load([]byte(defaultConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	mgmtConfig := b.Instance.Spec.Config.Mgmt
	if &mgmtConfig.Port != nil {
		cfg.Section("mgmt").Key("port").SetValue(fmt.Sprintf("%d", mgmtConfig.Port))
	}
	if &mgmtConfig.TlsPort != nil {
		cfg.Section("mgmt").Key("tls_port").SetValue(fmt.Sprintf("%d", mgmtConfig.TlsPort))
	}

	return cfg, nil
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
