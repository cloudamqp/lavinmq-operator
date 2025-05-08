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

var ConfigFileName = "lavinmq.ini"

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

[clustering]
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
	cfg, err := ini.Load([]byte(defaultConfig))

	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	b.AppendMainConfig(cfg)
	b.AppendAmqpConfig(cfg)
	b.AppendMqttConfig(cfg)
	b.AppendMgmtConfig(cfg)
	b.AppendClusteringConfig(cfg)

	_, err = cfg.WriteTo(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to write cluster config: %w", err)
	}

	configMap.Data[ConfigFileName] = config.String()
	return configMap, nil
}

func (b *ConfigReconciler) AppendMainConfig(cfg *ini.File) {
	mainConfig := b.Instance.Spec.Config.Main

	if mainConfig.ConsumerTimeout != 0 {
		cfg.Section("main").Key("consumer_timeout").SetValue(fmt.Sprintf("%d", mainConfig.ConsumerTimeout))
	}
	if mainConfig.DefaultConsumerPrefetch != 0 {
		cfg.Section("main").Key("default_consumer_prefetch").SetValue(fmt.Sprintf("%d", mainConfig.DefaultConsumerPrefetch))
	}
	if mainConfig.DefaultPassword != "" {
		cfg.Section("main").Key("default_password").SetValue(mainConfig.DefaultPassword)
	}
	if mainConfig.DefaultUser != "" {
		cfg.Section("main").Key("default_user").SetValue(mainConfig.DefaultUser)
	}
	if mainConfig.FreeDiskMin != 0 {
		cfg.Section("main").Key("free_disk_min").SetValue(fmt.Sprintf("%d", mainConfig.FreeDiskMin))
	}
	if mainConfig.FreeDiskWarn != 0 {
		cfg.Section("main").Key("free_disk_warn").SetValue(fmt.Sprintf("%d", mainConfig.FreeDiskWarn))
	}
	if mainConfig.LogExchange {
		cfg.Section("main").Key("log_exchange").SetValue(fmt.Sprintf("%t", mainConfig.LogExchange))
	}
	if mainConfig.LogLevel != "" {
		cfg.Section("main").Key("log_level").SetValue(mainConfig.LogLevel)
	}
	if mainConfig.MaxDeletedDefinitions != 0 {
		cfg.Section("main").Key("max_deleted_definitions").SetValue(fmt.Sprintf("%d", mainConfig.MaxDeletedDefinitions))
	}
	if mainConfig.SegmentSize != 0 {
		cfg.Section("main").Key("segment_size").SetValue(fmt.Sprintf("%d", mainConfig.SegmentSize))
	}
	if mainConfig.SetTimestamp {
		cfg.Section("main").Key("set_timestamp").SetValue(fmt.Sprintf("%t", mainConfig.SetTimestamp))
	}
	if mainConfig.SocketBufferSize != 0 {
		cfg.Section("main").Key("socket_buffer_size").SetValue(fmt.Sprintf("%d", mainConfig.SocketBufferSize))
	}
	if mainConfig.StatsInterval != 0 {
		cfg.Section("main").Key("stats_interval").SetValue(fmt.Sprintf("%d", mainConfig.StatsInterval))
	}
	if mainConfig.StatsLogSize != 0 {
		cfg.Section("main").Key("stats_log_size").SetValue(fmt.Sprintf("%d", mainConfig.StatsLogSize))
	}
	if mainConfig.TcpKeepalive != "" {
		cfg.Section("main").Key("tcp_keepalive").SetValue(mainConfig.TcpKeepalive)
	}
	if mainConfig.TcpNodelay {
		cfg.Section("main").Key("tcp_nodelay").SetValue(fmt.Sprintf("%t", mainConfig.TcpNodelay))
	}
	if mainConfig.TlsCiphers != "" {
		cfg.Section("main").Key("tls_ciphers").SetValue(mainConfig.TlsCiphers)
	}
	if mainConfig.TlsMinVersion != "" {
		cfg.Section("main").Key("tls_min_version").SetValue(mainConfig.TlsMinVersion)
	}
	if b.Instance.Spec.TlsSecret != nil {
		cfg.Section("main").Key("tls_cert").SetValue(fmt.Sprintf("/etc/lavinmq/tls/%s", "tls.crt"))
		cfg.Section("main").Key("tls_key").SetValue(fmt.Sprintf("/etc/lavinmq/tls/%s", "tls.key"))
	}
}

func (b *ConfigReconciler) AppendClusteringConfig(cfg *ini.File) {

	if b.Instance.Spec.EtcdEndpoints != nil {
		cfg.Section("clustering").Key("etcd_prefix").SetValue(b.Instance.Name)
		cfg.Section("clustering").Key("etcd_endpoints").SetValue(strings.Join(b.Instance.Spec.EtcdEndpoints, ","))
		cfg.Section("clustering").Key("enabled").SetValue("true")
	}

	if b.Instance.Spec.Config.Clustering.MaxUnsyncedActions != 0 {
		cfg.Section("clustering").Key("max_unsynced_actions").SetValue(fmt.Sprintf("%d", b.Instance.Spec.Config.Clustering.MaxUnsyncedActions))
	}
}

func (b *ConfigReconciler) AppendAmqpConfig(cfg *ini.File) {
	amqpConfig := b.Instance.Spec.Config.Amqp

	if amqpConfig.ChannelMax != 0 {
		cfg.Section("amqp").Key("channel_max").SetValue(fmt.Sprintf("%d", amqpConfig.ChannelMax))
	}
	if amqpConfig.FrameMax != 0 {
		cfg.Section("amqp").Key("frame_max").SetValue(fmt.Sprintf("%d", amqpConfig.FrameMax))
	}
	if amqpConfig.Heartbeat != 0 {
		cfg.Section("amqp").Key("heartbeat").SetValue(fmt.Sprintf("%d", amqpConfig.Heartbeat))
	}
	if amqpConfig.MaxMessageSize != 0 {
		cfg.Section("amqp").Key("max_message_size").SetValue(fmt.Sprintf("%d", amqpConfig.MaxMessageSize))
	}

	if amqpConfig.TlsPort != 0 {
		cfg.Section("amqp").Key("tls_port").SetValue(fmt.Sprintf("%d", amqpConfig.TlsPort))
	}

	cfg.Section("amqp").Key("port").SetValue(fmt.Sprintf("%d", amqpConfig.Port))
}

func (b *ConfigReconciler) AppendMqttConfig(cfg *ini.File) {
	mqttConfig := b.Instance.Spec.Config.Mqtt

	if mqttConfig.MaxInflightMessages != 0 {
		cfg.Section("mqtt").Key("max_inflight_messages").SetValue(fmt.Sprintf("%d", mqttConfig.MaxInflightMessages))
	}

	if mqttConfig.TlsPort != 0 {
		cfg.Section("mqtt").Key("tls_port").SetValue(fmt.Sprintf("%d", mqttConfig.TlsPort))
	}

	cfg.Section("mqtt").Key("port").SetValue(fmt.Sprintf("%d", mqttConfig.Port))
}

func (b *ConfigReconciler) AppendMgmtConfig(cfg *ini.File) {
	mgmtConfig := b.Instance.Spec.Config.Mgmt

	if mgmtConfig.TlsPort != 0 {
		cfg.Section("mgmt").Key("tls_port").SetValue(fmt.Sprintf("%d", mgmtConfig.TlsPort))
	}

	cfg.Section("mgmt").Key("port").SetValue(fmt.Sprintf("%d", mgmtConfig.Port))
}

func (b *ConfigReconciler) updateFields(_ context.Context, configMap *corev1.ConfigMap) error {
	newConfigMap, err := b.newObject()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(configMap.Data[ConfigFileName], newConfigMap.Data[ConfigFileName]) {
		configMap.Data[ConfigFileName] = newConfigMap.Data[ConfigFileName]
	}

	return nil
}

// Name returns the name of the config reconciler
func (b *ConfigReconciler) Name() string {
	return "config"
}
