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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"
	testutils "lavinmq-operator/internal/test_utils"
)

func TestNonExistentLavinMQ(t *testing.T) {
	t.Parallel()
	reconciler, lavinmq := setupResources(t)

	defer cleanupResources(t, lavinmq)

	result, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      lavinmq.Name,
			Namespace: lavinmq.Namespace,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, result, reconcile.Result{})
}

func TestDefaultLavinMQ(t *testing.T) {
	t.Parallel()
	_, lavinmq := setupResources(t)

	defer cleanupResources(t, lavinmq)

	err := k8sClient.Create(t.Context(), lavinmq)
	assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

	resource := &cloudamqpcomv1alpha1.LavinMQ{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{
		Name:      lavinmq.Name,
		Namespace: lavinmq.Namespace,
	}, resource)

	assert.NoErrorf(t, err, "Failed to get LavinMQ resource")

	assert.Equal(t, "cloudamqp/lavinmq:2.3.0", resource.Spec.Image)
	assert.Equal(t, int32(1), resource.Spec.Replicas)
}

func TestCreatingCustomLavinMQ(t *testing.T) {

	t.Run("Custom Port", func(t *testing.T) {
		t.Parallel()
		_, lavinmq := setupResources(t)

		defer cleanupResources(t, lavinmq)

		lavinmq.Spec.Config.Amqp.Port = 1337
		err := k8sClient.Create(t.Context(), lavinmq)

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      lavinmq.Name,
			Namespace: lavinmq.Namespace,
		}, resource)

		assert.NoErrorf(t, err, "Failed to get LavinMQ resource")
		assert.Equal(t, int32(1337), resource.Spec.Config.Amqp.Port)
	})
	t.Run("Custom Image", func(t *testing.T) {
		t.Parallel()
		_, lavinmq := setupResources(t)

		defer cleanupResources(t, lavinmq)

		lavinmq.Spec.Image = "cloudamqp/lavinmq:2.3.0"
		err := k8sClient.Create(t.Context(), lavinmq)

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      lavinmq.Name,
			Namespace: lavinmq.Namespace,
		}, resource)

		assert.NoErrorf(t, err, "Failed to get LavinMQ resource")

		assert.Equal(t, "cloudamqp/lavinmq:2.3.0", resource.Spec.Image)
	})

	t.Run("Custom Replicas", func(t *testing.T) {
		t.Parallel()
		_, lavinmq := setupResources(t)

		defer cleanupResources(t, lavinmq)

		lavinmq.Spec.Replicas = 3
		err := k8sClient.Create(t.Context(), lavinmq)

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      lavinmq.Name,
			Namespace: lavinmq.Namespace,
		}, resource)

		assert.NoErrorf(t, err, "Failed to get LavinMQ resource")

		assert.Equal(t, int32(3), resource.Spec.Replicas)
	})
}

func TestUpdatingLavinMQ(t *testing.T) {
	t.Run("Updating Ports", func(t *testing.T) {
		t.Parallel()
		reconciler, lavinmq := setupResources(t)

		defer cleanupResources(t, lavinmq)

		lavinmq.Spec.Config.Amqp.Port = 1337
		err := k8sClient.Create(t.Context(), lavinmq)

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		defer cleanupResources(t, lavinmq)

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      lavinmq.Name,
				Namespace: lavinmq.Namespace,
			},
		})

		assert.NoErrorf(t, err, "Failed to reconcile")

		lavinmq.Spec.Config.Amqp.Port = 1337
		err = k8sClient.Update(t.Context(), lavinmq)
		assert.NoErrorf(t, err, "Failed to update LavinMQ resource")

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      lavinmq.Name,
				Namespace: lavinmq.Namespace,
			},
		})

		assert.NoErrorf(t, err, "Failed to reconcile")

		resource := &appsv1.StatefulSet{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      lavinmq.Name,
			Namespace: lavinmq.Namespace,
		}, resource)

		assert.NoErrorf(t, err, "Failed to get StatefulSet")

		expectedPorts := []corev1.ContainerPort{
			{ContainerPort: 15672, Name: "http", Protocol: "TCP"},
			{ContainerPort: 1337, Name: "amqp", Protocol: "TCP"},
			{ContainerPort: 1883, Name: "mqtt", Protocol: "TCP"},
		}

		assert.Len(t, resource.Spec.Template.Spec.Containers[0].Ports, len(expectedPorts))

		for i, port := range expectedPorts {
			assert.Equal(t, port, resource.Spec.Template.Spec.Containers[0].Ports[i])
		}
	})

	t.Run("Updating Image", func(t *testing.T) {
		t.Parallel()
		reconciler, lavinmq := setupResources(t)

		defer cleanupResources(t, lavinmq)

		lavinmq.Spec.Image = "cloudamqp/lavinmq:2.2.0"
		err := k8sClient.Create(t.Context(), lavinmq)

		assert.NoErrorf(t, err, "Failed to create LavinMQ resource")

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      lavinmq.Name,
				Namespace: lavinmq.Namespace,
			},
		})

		assert.NoErrorf(t, err, "Failed to reconcile")

		lavinmq.Spec.Image = "cloudamqp/lavinmq:2.3.0"
		err = k8sClient.Update(t.Context(), lavinmq)
		assert.NoErrorf(t, err, "Failed to update LavinMQ resource")

		_, err = reconciler.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      lavinmq.Name,
				Namespace: lavinmq.Namespace,
			},
		})

		assert.NoErrorf(t, err, "Failed to reconcile")

		resource := &appsv1.StatefulSet{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      lavinmq.Name,
			Namespace: lavinmq.Namespace,
		}, resource)

		assert.NoErrorf(t, err, "Failed to get StatefulSet")

		assert.Equal(t, "cloudamqp/lavinmq:2.3.0", resource.Spec.Template.Spec.Containers[0].Image)
	})

}

func setupResources(t *testing.T) (*LavinMQReconciler, *cloudamqpcomv1alpha1.LavinMQ) {
	reconciler := &LavinMQReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}

	lavinmq := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	err := testutils.CreateNamespace(t.Context(), k8sClient, lavinmq.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
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

	err = testutils.DeleteNamespace(t.Context(), k8sClient, namespace)
	assert.NoErrorf(t, err, "Failed to delete namespace")
}
