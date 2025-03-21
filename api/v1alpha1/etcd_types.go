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

// EtcdSpec defines the desired state of Etcd
type EtcdSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:default="bitnami/etcd:3.5.19"
	// +optional
	Image string `json:"image,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:default={{containerPort:2379,name:"client"},{containerPort:2380,name:"peer"}}
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`

	// +required
	DataVolumeClaimSpec corev1.PersistentVolumeClaimSpec `json:"dataVolumeClaim"`
}

// EtcdStatus defines the observed state of Etcd
type EtcdStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Etcd is the Schema for the etcds API
type Etcd struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EtcdSpec   `json:"spec,omitempty"`
	Status EtcdStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EtcdList contains a list of Etcd
type EtcdList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Etcd `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Etcd{}, &EtcdList{})
}
