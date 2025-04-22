/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LavinMQSpec defines the desired state of LavinMQ
type LavinMQSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:default="cloudamqp/lavinmq:2.2.0"
	// +optional
	Image string `json:"image,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Will override the accessmode and force it to ReadWriteOnce
	// +required
	DataVolumeClaimSpec corev1.PersistentVolumeClaimSpec `json:"dataVolumeClaim"`

	// +optional
	EtcdEndpoints []string `json:"etcdEndpoints,omitempty"`

	// +optional
	TlsSecret *corev1.SecretReference `json:"tlsSecret,omitempty"`

	// +optional
	Config LavinMQConfig `json:"config,omitempty"`
}

type MainConfig struct {
	// The timeout for consumers in milliseconds.
	// +optional
	ConsumerTimeout uint64 `json:"consumer_timeout,omitempty"`

	// Default prefetch value for consumers if not set by the consumer.
	// +optional
	DefaultConsumerPrefetch uint64 `json:"default_consumer_prefetch,omitempty"`

	// Hashed password for the default user.
	// Use lavinmqctl hash_password or /api/auth/hash_password to generate the password hash.
	// +optional
	DefaultPassword string `json:"default_password,omitempty"`

	// The default user.
	// +optional
	DefaultUser string `json:"default_user,omitempty"`

	// The minimum value of free disk space in bytes before LavinMQ starts to control flow.
	// +optional
	FreeDiskMin uint64 `json:"free_disk_min,omitempty"`

	// The minimum value of free disk space in bytes before LavinMQ warns about low disk space.
	// +optional
	FreeDiskWarn uint64 `json:"free_disk_warn,omitempty"`

	// Enables the log exchange.
	// +optional
	LogExchange bool `json:"log_exchange,omitempty"`

	// Controls how detailed the log should be.
	// The level can be one of: none, fatal, error, warn, info, debug.
	// +optional
	LogLevel string `json:"log_level,omitempty"`

	// The number of deleted queues, unbinds, etc., that compacts the definitions file.
	// +optional
	MaxDeletedDefinitions uint64 `json:"max_deleted_definitions,omitempty"`

	// The size of segment files in bytes.
	// +optional
	SegmentSize uint64 `json:"segment_size,omitempty"`

	// Enables setting the timestamp property in msg headers.
	// +optional
	SetTimestamp bool `json:"set_timestamp,omitempty"`

	// The socket buffer size in bytes.
	// +optional
	SocketBufferSize uint64 `json:"socket_buffer_size,omitempty"`

	// Statistics collection interval in milliseconds.
	// +optional
	StatsInterval uint64 `json:"stats_interval,omitempty"`

	// Number of entries in the statistics log file before the oldest entry is removed.
	// +optional
	StatsLogSize uint64 `json:"stats_log_size,omitempty"`

	// TCP keepalive settings as a tuple {idle, interval, probes/count}.
	// +optional
	TcpKeepalive string `json:"tcp_keepalive,omitempty"`

	// Setting for disabling Nagle's algorithm and sending the data as soon as it's available.
	// +optional
	TcpNodelay bool `json:"tcp_nodelay,omitempty"`

	// Specifies the TLS ciphers to use.
	// +optional
	TlsCiphers string `json:"tls_ciphers,omitempty"`

	// Specifies the minimum TLS version to use.
	// +optional
	TlsMinVersion string `json:"tls_min_version,omitempty"`
}

type MgmtConfig struct {
	// Port for the HTTP management interface.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=15672
	// +optional
	Port int32 `json:"port,omitempty"`

	// Port for the HTTPS management interface.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +optional
	TlsPort int32 `json:"tls_port,omitempty"`
}

type AmqpConfig struct {
	// Maximum number of channels per connection.
	// +optional
	ChannelMax uint64 `json:"channel_max,omitempty"`

	// Maximum size of an AMQP frame in bytes.
	// +optional
	FrameMax uint64 `json:"frame_max,omitempty"`

	// Interval in seconds for AMQP heartbeats.
	// +optional
	Heartbeat uint64 `json:"heartbeat,omitempty"`

	// Maximum size of a message in bytes.
	// +optional
	MaxMessageSize uint64 `json:"max_message_size,omitempty"`

	// Port for the AMQP interface.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=5672
	// +optional
	Port int32 `json:"port,omitempty"`

	// Port for the AMQPS interface.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +optional
	TlsPort int32 `json:"tls_port,omitempty"`
}

type MqttConfig struct {
	// Maximum number of in-flight messages per client.
	// +optional
	MaxInflightMessages uint64 `json:"max_inflight_messages,omitempty"`

	// Port for the MQTT interface.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=1883
	// +optional
	Port int32 `json:"port,omitempty"`

	// Port for the MQTTS interface.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +optional
	TlsPort int32 `json:"tls_port,omitempty"`
}

type ClusteringConfig struct {
	// Maximum number of unsynced actions allowed in the cluster.
	// +optional
	MaxUnsyncedActions uint64 `json:"max_unsynced_actions,omitempty"`
}

type LavinMQConfig struct {
	Main       MainConfig       `json:"main,omitempty"`
	Mgmt       MgmtConfig       `json:"mgmt,omitempty"`
	Amqp       AmqpConfig       `json:"amqp,omitempty"`
	Mqtt       MqttConfig       `json:"mqtt,omitempty"`
	Clustering ClusteringConfig `json:"clustering,omitempty"`
}

// LavinMQStatus defines the observed state of LavinMQ
type LavinMQStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions store the status conditions of the LavinMQ instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// LavinMQ is the Schema for the lavinmqs API
type LavinMQ struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LavinMQSpec   `json:"spec,omitempty"`
	Status LavinMQStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LavinMQList contains a list of LavinMQ
type LavinMQList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LavinMQ `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LavinMQ{}, &LavinMQList{})
}
