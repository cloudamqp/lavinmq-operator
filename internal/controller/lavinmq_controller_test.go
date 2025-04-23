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

package controller

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"
)

func TestNonExistentLavinMQ(t *testing.T) {
	reconciler, lavinmq := setupResources()

	defer cleanupResources(t, lavinmq)

	result, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-resource",
			Namespace: "default",
		},
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != (reconcile.Result{}) {
		t.Errorf("Expected empty result, got %v", result)
	}
}

func TestDefaultLavinMQ(t *testing.T) {
	_, lavinmq := setupResources()

	defer cleanupResources(t, lavinmq)

	err := k8sClient.Create(t.Context(), lavinmq)
	if err != nil {
		t.Errorf("Failed to create LavinMQ resource: %v", err)
	}

	resource := &cloudamqpcomv1alpha1.LavinMQ{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{
		Name:      "test-resource",
		Namespace: "default",
	}, resource)

	if err != nil {
		t.Fatalf("Failed to get LavinMQ resource: %v", err)
	}

	if resource.Spec.Image != "cloudamqp/lavinmq:2.2.0" {
		t.Errorf("Expected image 'cloudamqp/lavinmq:2.2.0', got '%s'", resource.Spec.Image)
	}

	if resource.Spec.Replicas != 1 {
		t.Errorf("Expected replicas 1, got %d", resource.Spec.Replicas)
	}
}

func TestCreatingCustomLavinMQ(t *testing.T) {
	_, lavinmq := setupResources()

	defer cleanupResources(t, lavinmq)

	t.Run("Custom Port", func(t *testing.T) {
		copy := lavinmq.DeepCopy()
		copy.Spec.Config.Amqp.Port = 1337
		copy.Name = "test-resource-custom-port"
		err := k8sClient.Create(t.Context(), copy)

		defer cleanupResources(t, copy)

		if err != nil {
			t.Errorf("Failed to create LavinMQ resource: %v", err)
		}

		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      copy.Name,
			Namespace: copy.Namespace,
		}, resource)
		if err != nil {
			t.Errorf("Failed to get LavinMQ resource: %v", err)
		}

		if resource.Spec.Config.Amqp.Port != 1337 {
			t.Errorf("Expected port 1337, got %d", resource.Spec.Config.Amqp.Port)
		}
	})
	t.Run("Custom Image", func(t *testing.T) {
		copy := lavinmq.DeepCopy()
		copy.Spec.Image = "cloudamqp/lavinmq:2.3.0"
		copy.Name = "test-resource-custom-image"
		err := k8sClient.Create(t.Context(), copy)

		defer cleanupResources(t, copy)

		if err != nil {
			t.Errorf("Failed to create LavinMQ resource: %v", err)
		}

		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      copy.Name,
			Namespace: copy.Namespace,
		}, resource)
		if err != nil {
			t.Fatalf("Failed to get LavinMQ resource: %v", err)
		}

		if resource.Spec.Image != "cloudamqp/lavinmq:2.3.0" {
			t.Errorf("Expected image 'cloudamqp/lavinmq:2.3.0', got '%s'", resource.Spec.Image)
		}
	})

	t.Run("Custom Replicas", func(t *testing.T) {
		copy := lavinmq.DeepCopy()
		copy.Spec.Replicas = 3
		copy.Name = "test-resource-custom-replicas"
		err := k8sClient.Create(t.Context(), copy)

		defer cleanupResources(t, copy)

		if err != nil {
			t.Errorf("Failed to create LavinMQ resource: %v", err)
		}

		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      copy.Name,
			Namespace: copy.Namespace,
		}, resource)

		if err != nil {
			t.Fatalf("Failed to get LavinMQ resource: %v", err)
		}

		if resource.Spec.Replicas != 3 {
			t.Errorf("Expected replicas 3, got %d", resource.Spec.Replicas)
		}
	})
}

func TestUpdatingLavinMQ(t *testing.T) {
	reconciler, lavinmq := setupResources()

	defer cleanupResources(t, lavinmq)

	t.Run("Updating Ports", func(t *testing.T) {
		copy := lavinmq.DeepCopy()
		copy.Name = "test-resource-updating-port"
		err := k8sClient.Create(t.Context(), copy)

		if err != nil {
			t.Errorf("Failed to create LavinMQ resource: %v", err)
		}

		defer cleanupResources(t, copy)

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      copy.Name,
				Namespace: copy.Namespace,
			},
		})

		if err != nil {
			t.Errorf("Failed to reconcile: %v", err)
		}

		copy.Spec.Config.Amqp.Port = 1337
		if err := k8sClient.Update(t.Context(), copy); err != nil {
			t.Errorf("Failed to update LavinMQ resource: %v", err)
		}

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      copy.Name,
				Namespace: copy.Namespace,
			},
		})

		if err != nil {
			t.Errorf("Failed to reconcile: %v", err)
		}

		resource := &appsv1.StatefulSet{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      copy.Name,
			Namespace: copy.Namespace,
		}, resource)
		if err != nil {
			t.Errorf("Failed to get StatefulSet: %v", err)
		}

		expectedPorts := []corev1.ContainerPort{
			{ContainerPort: 15672, Name: "http", Protocol: "TCP"},
			{ContainerPort: 1337, Name: "amqp", Protocol: "TCP"},
			{ContainerPort: 1883, Name: "mqtt", Protocol: "TCP"},
		}

		if len(resource.Spec.Template.Spec.Containers[0].Ports) != len(expectedPorts) {
			t.Errorf("Expected %d ports, got %d", len(expectedPorts), len(resource.Spec.Template.Spec.Containers[0].Ports))
		}

		for i, port := range expectedPorts {
			if resource.Spec.Template.Spec.Containers[0].Ports[i] != port {
				t.Errorf("Expected port %v, got %v", port, resource.Spec.Template.Spec.Containers[0].Ports[i])
			}
		}
	})

	t.Run("Updating Image", func(t *testing.T) {
		copy := lavinmq.DeepCopy()
		copy.Name = "test-resource-updating-image"
		err := k8sClient.Create(t.Context(), copy)

		defer cleanupResources(t, copy)

		if err != nil {
			t.Errorf("Failed to create LavinMQ resource: %v", err)
		}

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      copy.Name,
				Namespace: copy.Namespace,
			},
		})

		if err != nil {
			t.Errorf("Failed to reconcile: %v", err)
		}

		copy.Spec.Image = "cloudamqp/lavinmq:2.3.0"
		if err := k8sClient.Update(t.Context(), copy); err != nil {
			t.Errorf("Failed to update LavinMQ resource: %v", err)
		}

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      copy.Name,
				Namespace: copy.Namespace,
			},
		})

		if err != nil {
			t.Errorf("Failed to reconcile: %v", err)
		}

		resource := &appsv1.StatefulSet{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      copy.Name,
			Namespace: copy.Namespace,
		}, resource)

		if err != nil {
			t.Errorf("Failed to get StatefulSet: %v", err)
		}

		if resource.Spec.Template.Spec.Containers[0].Image != "cloudamqp/lavinmq:2.3.0" {
			t.Errorf("Expected image 'cloudamqp/lavinmq:2.3.0', got '%s'", resource.Spec.Template.Spec.Containers[0].Image)
		}
	})

}

func setupResources() (*LavinMQReconciler, *cloudamqpcomv1alpha1.LavinMQ) {
	reconciler := &LavinMQReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}

	lavinmq := &cloudamqpcomv1alpha1.LavinMQ{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
		Spec: cloudamqpcomv1alpha1.LavinMQSpec{
			DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		},
	}

	return reconciler, lavinmq
}

func cleanupResources(t *testing.T, lavinmq *cloudamqpcomv1alpha1.LavinMQ) {
	resourceName := lavinmq.Name
	namespace := lavinmq.Namespace

	// Clean up StatefulSet
	sts := &appsv1.StatefulSet{}
	err := k8sClient.Get(t.Context(), types.NamespacedName{
		Name:      resourceName,
		Namespace: namespace,
	}, sts)
	if err == nil {
		if err := k8sClient.Delete(t.Context(), sts); err != nil {
			t.Errorf("Failed to delete StatefulSet: %v", err)
		}
	}

	// Clean up ConfigMap
	configMap := &corev1.ConfigMap{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{
		Name:      fmt.Sprintf("%s-config", resourceName),
		Namespace: namespace,
	}, configMap)
	if err == nil {
		if err := k8sClient.Delete(t.Context(), configMap); err != nil {
			t.Errorf("Failed to delete ConfigMap: %v", err)
		}
	}

	// Clean up LavinMQ
	resource := &cloudamqpcomv1alpha1.LavinMQ{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{
		Name:      resourceName,
		Namespace: namespace,
	}, resource)
	if err == nil {
		if err := k8sClient.Delete(t.Context(), resource); err != nil {
			t.Errorf("Failed to delete LavinMQ resource: %v", err)
		}
	}
}
