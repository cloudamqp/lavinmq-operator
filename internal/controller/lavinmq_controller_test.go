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

	"github.com/stretchr/testify/assert"
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

	assert.NoError(t, err)
	assert.Equal(t, result, reconcile.Result{})
}

func TestDefaultLavinMQ(t *testing.T) {
	_, lavinmq := setupResources()

	defer cleanupResources(t, lavinmq)

	err := k8sClient.Create(t.Context(), lavinmq)
	assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

	resource := &cloudamqpcomv1alpha1.LavinMQ{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{
		Name:      "test-resource",
		Namespace: "default",
	}, resource)

	assert.NoErrorf(t, err, "Failed to get LavinMQ resource")

	assert.Equal(t, resource.Spec.Image, "cloudamqp/lavinmq:2.2.0")
	assert.Equal(t, resource.Spec.Replicas, int32(1))
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

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      copy.Name,
			Namespace: copy.Namespace,
		}, resource)

		assert.NoErrorf(t, err, "Failed to get LavinMQ resource")
		assert.Equal(t, resource.Spec.Config.Amqp.Port, int32(1337))
	})
	t.Run("Custom Image", func(t *testing.T) {
		copy := lavinmq.DeepCopy()
		copy.Spec.Image = "cloudamqp/lavinmq:2.3.0"
		copy.Name = "test-resource-custom-image"
		err := k8sClient.Create(t.Context(), copy)

		defer cleanupResources(t, copy)

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      copy.Name,
			Namespace: copy.Namespace,
		}, resource)

		assert.NoErrorf(t, err, "Failed to get LavinMQ resource")

		assert.Equal(t, resource.Spec.Image, "cloudamqp/lavinmq:2.3.0")
	})

	t.Run("Custom Replicas", func(t *testing.T) {
		copy := lavinmq.DeepCopy()
		copy.Spec.Replicas = 3
		copy.Name = "test-resource-custom-replicas"
		err := k8sClient.Create(t.Context(), copy)

		defer cleanupResources(t, copy)

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      copy.Name,
			Namespace: copy.Namespace,
		}, resource)

		assert.NoErrorf(t, err, "Failed to get LavinMQ resource")

		assert.Equal(t, resource.Spec.Replicas, int32(3))
	})
}

func TestUpdatingLavinMQ(t *testing.T) {
	reconciler, lavinmq := setupResources()

	defer cleanupResources(t, lavinmq)

	t.Run("Updating Ports", func(t *testing.T) {
		copy := lavinmq.DeepCopy()
		copy.Name = "test-resource-updating-port"
		err := k8sClient.Create(t.Context(), copy)

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		defer cleanupResources(t, copy)

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      copy.Name,
				Namespace: copy.Namespace,
			},
		})

		assert.NoErrorf(t, err, "Failed to reconcile")

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

		assert.Len(t, resource.Spec.Template.Spec.Containers[0].Ports, len(expectedPorts))

		for i, port := range expectedPorts {
			assert.Equal(t, resource.Spec.Template.Spec.Containers[0].Ports[i], port)
		}
	})

	t.Run("Updating Image", func(t *testing.T) {
		copy := lavinmq.DeepCopy()
		copy.Name = "test-resource-updating-image"
		err := k8sClient.Create(t.Context(), copy)

		defer cleanupResources(t, copy)

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      copy.Name,
				Namespace: copy.Namespace,
			},
		})

		assert.NoErrorf(t, err, "Failed to reconcile")

		copy.Spec.Image = "cloudamqp/lavinmq:2.3.0"
		err = k8sClient.Update(t.Context(), copy)
		assert.NoErrorf(t, err, "Failed to update LavinMQ resource")

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      copy.Name,
				Namespace: copy.Namespace,
			},
		})

		assert.NoErrorf(t, err, "Failed to reconcile")

		resource := &appsv1.StatefulSet{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      copy.Name,
			Namespace: copy.Namespace,
		}, resource)

		assert.NoErrorf(t, err, "Failed to get StatefulSet")

		assert.Equal(t, resource.Spec.Template.Spec.Containers[0].Image, "cloudamqp/lavinmq:2.3.0")
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
		err = k8sClient.Delete(t.Context(), sts)
		assert.NoErrorf(t, err, "Failed to delete StatefulSet")
	}

	// Clean up ConfigMap
	configMap := &corev1.ConfigMap{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{
		Name:      fmt.Sprintf("%s-config", resourceName),
		Namespace: namespace,
	}, configMap)
	if err == nil {
		err = k8sClient.Delete(t.Context(), configMap)
		assert.NoErrorf(t, err, "Failed to delete ConfigMap")
	}

	// Clean up LavinMQ
	resource := &cloudamqpcomv1alpha1.LavinMQ{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{
		Name:      resourceName,
		Namespace: namespace,
	}, resource)
	if err == nil {
		err = k8sClient.Delete(t.Context(), resource)
		assert.NoErrorf(t, err, "Failed to delete LavinMQ resource")
	}
}
