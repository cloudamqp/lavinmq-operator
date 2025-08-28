package reconciler_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	cloudamqpcomv1alpha1 "github.com/cloudamqp/lavinmq-operator/api/v1alpha1"
	"github.com/cloudamqp/lavinmq-operator/internal/reconciler"
	testutils "github.com/cloudamqp/lavinmq-operator/internal/test_utils"
)

func TestStatefulSetReconciler(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})

	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

	configMap := createConfigMap(t, instance, "initial_config")
	defer deleteConfigMap(t, configMap)

	rc := &reconciler.StatefulSetReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	err = k8sClient.Create(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to create instance")

	instance.Spec.Image = "test-image:latest2"
	err = k8sClient.Update(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to update instance")

	_, err = rc.Reconcile(t.Context())
	assert.NoErrorf(t, err, "Failed to reconcile instance")

	sts := &appsv1.StatefulSet{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sts)
	assert.NoErrorf(t, err, "Failed to get statefulset")

	assert.Equal(t, "test-image:latest2", sts.Spec.Template.Spec.Containers[0].Image)
}

func TestCreateContainerResources(t *testing.T) {
	t.Parallel()
	Resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("2566Mi"),
		},
	}

	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{
		Resources: Resources,
	})

	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

	configMap := createConfigMap(t, instance, "initial_config")
	defer deleteConfigMap(t, configMap)

	rc := &reconciler.StatefulSetReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	err = k8sClient.Create(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to create instance")

	_, err = rc.Reconcile(t.Context())
	assert.NoErrorf(t, err, "Failed to reconcile instance")

	sts := &appsv1.StatefulSet{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sts)
	assert.NoErrorf(t, err, "Failed to get statefulset")

	assert.True(t, reflect.DeepEqual(Resources, sts.Spec.Template.Spec.Containers[0].Resources))
}

func TestUpdateContainerResources(t *testing.T) {
	t.Parallel()
	Resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("2566Mi"),
		},
	}

	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{
		Resources: Resources,
	})

	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

	configMap := createConfigMap(t, instance, "initial_config")
	defer deleteConfigMap(t, configMap)

	rc := &reconciler.StatefulSetReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	err = k8sClient.Create(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to create instance")

	// Reconciler is creating the sts
	_, err = rc.Reconcile(t.Context())
	assert.NoErrorf(t, err, "Failed to reconcile instance")

	// Reconciler is updating the sts
	Resources.Limits[corev1.ResourceCPU] = resource.MustParse("500m")
	Resources.Requests[corev1.ResourceCPU] = resource.MustParse("300m")
	instance.Spec.Resources = Resources

	err = k8sClient.Update(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to update instance")
	_, err = rc.Reconcile(t.Context())
	assert.NoErrorf(t, err, "Failed to reconcile instance")

	sts := &appsv1.StatefulSet{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sts)
	assert.NoErrorf(t, err, "Failed to get statefulset")

	assert.True(t, reflect.DeepEqual(Resources, sts.Spec.Template.Spec.Containers[0].Resources))
}

func TestStsNodeSelector(t *testing.T) {
	t.Parallel()
	selector := map[string]string{"disktype": "ssd"}
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})
	instance.Spec.NodeSelector = selector

	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

	configMap := createConfigMap(t, instance, "initial_config")
	defer deleteConfigMap(t, configMap)

	rc := &reconciler.StatefulSetReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}
	err = k8sClient.Create(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to create instance")

	// Reconciler is creating the sts
	_, err = rc.Reconcile(t.Context())
	assert.NoErrorf(t, err, "Failed to reconcile instance")

	sts := &appsv1.StatefulSet{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sts)
	assert.NoErrorf(t, err, "Failed to get statefulset")

	assert.True(t, reflect.DeepEqual(selector, sts.Spec.Template.Spec.NodeSelector))
}

func TestConfigHashAnnotation(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})

	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

	configMap := createConfigMap(t, instance, "initial_config")
	defer deleteConfigMap(t, configMap)

	rc := &reconciler.StatefulSetReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	// Create instance and reconcile
	err = k8sClient.Create(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to create instance")

	_, err = rc.Reconcile(t.Context())
	assert.NoErrorf(t, err, "Failed to reconcile instance")

	// Get StatefulSet and verify initial hash
	sts := &appsv1.StatefulSet{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sts)
	assert.NoErrorf(t, err, "Failed to get statefulset")

	initialHash := sts.Spec.Template.ObjectMeta.Annotations["config-hash"]
	assert.NotEmpty(t, initialHash, "Config hash annotation should be set")

	// Update ConfigMap
	configMap.Data[reconciler.ConfigFileName] = "updated_config"
	err = k8sClient.Update(t.Context(), configMap)
	assert.NoErrorf(t, err, "Failed to update ConfigMap")

	// Reconcile again
	_, err = rc.Reconcile(t.Context())
	assert.NoErrorf(t, err, "Failed to reconcile instance")

	// Get updated StatefulSet and verify hash changed
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sts)
	assert.NoErrorf(t, err, "Failed to get updated statefulset")

	updatedHash := sts.Spec.Template.ObjectMeta.Annotations["config-hash"]
	assert.NotEmpty(t, updatedHash, "Config hash annotation should still be set")
	assert.NotEqual(t, initialHash, updatedHash, "Config hash should change when ConfigMap content changes")
}

func createConfigMap(t *testing.T, instance *cloudamqpcomv1alpha1.LavinMQ, config string) *corev1.ConfigMap {
	// Create initial ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Data: map[string]string{
			reconciler.ConfigFileName: config,
		},
	}
	err := k8sClient.Create(t.Context(), configMap)
	assert.NoErrorf(t, err, "Failed to create initial ConfigMap")

	return configMap
}

func deleteConfigMap(t *testing.T, configMap *corev1.ConfigMap) {
	err := k8sClient.Delete(t.Context(), configMap)
	assert.NoErrorf(t, err, "Failed to delete ConfigMap")
}
