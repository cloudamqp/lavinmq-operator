package reconciler

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudamqp/lavinmq-operator/internal/controller/utils"

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
	b.AppendSniConfig(cfg)

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

func (b *ConfigReconciler) AppendSniConfig(cfg *ini.File) {
	for _, sniConfig := range b.Instance.Spec.Config.Sni {
		sectionName := fmt.Sprintf("sni:%s", sniConfig.Hostname)

		// Base TLS configuration - always using mounted secrets
		certPath := fmt.Sprintf("/etc/lavinmq/sni/%s/tls.crt", sniConfig.Hostname)
		keyPath := fmt.Sprintf("/etc/lavinmq/sni/%s/tls.key", sniConfig.Hostname)
		cfg.Section(sectionName).Key("tls_cert").SetValue(certPath)
		cfg.Section(sectionName).Key("tls_key").SetValue(keyPath)

		// CA certificate for mTLS (if provided)
		if sniConfig.TlsCaSecret != nil {
			caPath := fmt.Sprintf("/etc/lavinmq/sni/%s-ca/ca.crt", sniConfig.Hostname)
			cfg.Section(sectionName).Key("tls_ca_cert").SetValue(caPath)
		}

		if sniConfig.TlsVerifyPeer {
			cfg.Section(sectionName).Key("tls_verify_peer").SetValue("true")
		}

		// Protocol-specific overrides for AMQP
		if sniConfig.Amqp != nil {
			if sniConfig.Amqp.TlsSecret != nil {
				amqpCertPath := fmt.Sprintf("/etc/lavinmq/sni/%s-amqp/tls.crt", sniConfig.Hostname)
				amqpKeyPath := fmt.Sprintf("/etc/lavinmq/sni/%s-amqp/tls.key", sniConfig.Hostname)
				cfg.Section(sectionName).Key("amqp_tls_cert").SetValue(amqpCertPath)
				cfg.Section(sectionName).Key("amqp_tls_key").SetValue(amqpKeyPath)
			}
			if sniConfig.Amqp.TlsCaSecret != nil {
				amqpCaPath := fmt.Sprintf("/etc/lavinmq/sni/%s-amqp-ca/ca.crt", sniConfig.Hostname)
				cfg.Section(sectionName).Key("amqp_tls_ca_cert").SetValue(amqpCaPath)
			}
			if sniConfig.Amqp.TlsVerifyPeer != nil {
				cfg.Section(sectionName).Key("amqp_tls_verify_peer").SetValue(fmt.Sprintf("%t", *sniConfig.Amqp.TlsVerifyPeer))
			}
		}

		// Protocol-specific overrides for MQTT
		if sniConfig.Mqtt != nil {
			if sniConfig.Mqtt.TlsSecret != nil {
				mqttCertPath := fmt.Sprintf("/etc/lavinmq/sni/%s-mqtt/tls.crt", sniConfig.Hostname)
				mqttKeyPath := fmt.Sprintf("/etc/lavinmq/sni/%s-mqtt/tls.key", sniConfig.Hostname)
				cfg.Section(sectionName).Key("mqtt_tls_cert").SetValue(mqttCertPath)
				cfg.Section(sectionName).Key("mqtt_tls_key").SetValue(mqttKeyPath)
			}
			if sniConfig.Mqtt.TlsCaSecret != nil {
				mqttCaPath := fmt.Sprintf("/etc/lavinmq/sni/%s-mqtt-ca/ca.crt", sniConfig.Hostname)
				cfg.Section(sectionName).Key("mqtt_tls_ca_cert").SetValue(mqttCaPath)
			}
			if sniConfig.Mqtt.TlsVerifyPeer != nil {
				cfg.Section(sectionName).Key("mqtt_tls_verify_peer").SetValue(fmt.Sprintf("%t", *sniConfig.Mqtt.TlsVerifyPeer))
			}
		}

		// Protocol-specific overrides for HTTP
		if sniConfig.Http != nil {
			if sniConfig.Http.TlsSecret != nil {
				httpCertPath := fmt.Sprintf("/etc/lavinmq/sni/%s-http/tls.crt", sniConfig.Hostname)
				httpKeyPath := fmt.Sprintf("/etc/lavinmq/sni/%s-http/tls.key", sniConfig.Hostname)
				cfg.Section(sectionName).Key("http_tls_cert").SetValue(httpCertPath)
				cfg.Section(sectionName).Key("http_tls_key").SetValue(httpKeyPath)
			}
			if sniConfig.Http.TlsCaSecret != nil {
				httpCaPath := fmt.Sprintf("/etc/lavinmq/sni/%s-http-ca/ca.crt", sniConfig.Hostname)
				cfg.Section(sectionName).Key("http_tls_ca_cert").SetValue(httpCaPath)
			}
			if sniConfig.Http.TlsVerifyPeer != nil {
				cfg.Section(sectionName).Key("http_tls_verify_peer").SetValue(fmt.Sprintf("%t", *sniConfig.Http.TlsVerifyPeer))
			}
		}
	}
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
