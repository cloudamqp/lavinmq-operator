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

	// +kubebuilder:default={{containerPort:5672,name:"amqp"},{containerPort:15672,name:"http"},{containerPort:1883,name:"mqtt"}}
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`

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
	// +optional
	ConsumerTimeout uint64 `json:"consumer_timeout,omitempty"`

	// +optional
	DefaultConsumerPrefetch uint64 `json:"default_consumer_prefetch,omitempty"`

	// +optional
	DefaultPassword string `json:"default_password,omitempty"`

	// +optional
	DefaultUser string `json:"default_user,omitempty"`

	// +optional
	FreeDiskMin uint64 `json:"free_disk_min,omitempty"`

	// +optional
	FreeDiskWarn uint64 `json:"free_disk_warn,omitempty"`

	// +optional
	LogExchange bool `json:"log_exchange,omitempty"`

	// +optional
	LogLevel string `json:"log_level,omitempty"`

	// +optional
	MaxDeletedDefinitions uint64 `json:"max_deleted_definitions,omitempty"`

	// +optional
	SegmentSize uint64 `json:"segment_size,omitempty"`

	// +optional
	SetTimestamp bool `json:"set_timestamp,omitempty"`

	// +optional
	SocketBufferSize uint64 `json:"socket_buffer_size,omitempty"`

	// +optional
	StatsInterval uint64 `json:"stats_interval,omitempty"`

	// +optional
	StatsLogSize uint64 `json:"stats_log_size,omitempty"`

	// +optional
	TcpKeepalive string `json:"tcp_keepalive,omitempty"`

	// +optional
	TcpNodelay bool `json:"tcp_nodelay,omitempty"`

	// +optional
	TlsCiphers string `json:"tls_ciphers,omitempty"`

	// +optional
	TlsMinVersion string `json:"tls_min_version,omitempty"`
}
type MgmtConfig struct {
	// +optional
	Port uint64 `json:"port,omitempty"`

	// +optional
	TlsPort uint64 `json:"tls_port,omitempty"`
}

type AmqpConfig struct {
	// +optional
	ChannelMax uint64 `json:"channel_max,omitempty"`
	// +optional
	FrameMax uint64 `json:"frame_max,omitempty"`
	// +optional
	Heartbeat uint64 `json:"heartbeat,omitempty"`
	// +optional
	MaxMessageSize uint64 `json:"max_message_size,omitempty"`
	// +optional
	Port uint64 `json:"port,omitempty"`
	// +optional
	TlsPort uint64 `json:"tls_port,omitempty"`
}

type MqttConfig struct {
	// +optional
	MaxInflightMessages uint64 `json:"max_inflight_messages,omitempty"`
	// +optional
	Port uint64 `json:"port,omitempty"`
	// +optional
	TlsPort uint64 `json:"tls_port,omitempty"`
}

type ClusteringConfig struct {
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
