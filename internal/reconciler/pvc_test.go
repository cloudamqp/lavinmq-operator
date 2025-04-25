package reconciler_test

import (
	"fmt"
	"lavinmq-operator/api/v1alpha1"
	"lavinmq-operator/internal/reconciler"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestDefaultPVCReconciler(t *testing.T) {
	instance := defaultInstance.DeepCopy()
	rc := &reconciler.PVCReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	err := k8sClient.Create(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to create instance")

	defer cleanupPvcResources(t, instance)

	rc.Reconcile(t.Context())

	pvc := &corev1.PersistentVolumeClaim{}
	assert.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc))
	assert.NotNil(t, pvc)

	assert.Equal(t, pvc.Name, fmt.Sprintf("data-%s-0", instance.Name))
	assert.Equal(t, pvc.Namespace, instance.Namespace)
	assert.Equal(t, pvc.Spec.AccessModes, instance.Spec.DataVolumeClaimSpec.AccessModes)
	// No diff
	assert.Zero(t, pvc.Spec.Resources.Requests.Storage().Cmp(*instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage()))
}

func TestNoChangesToPVC(t *testing.T) {
	instance := defaultInstance.DeepCopy()
	rc := &reconciler.PVCReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	err := k8sClient.Create(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to create instance")

	defer cleanupPvcResources(t, instance)

	_, err = rc.Reconcile(t.Context())
	assert.NoError(t, err)

	createdPvc := &corev1.PersistentVolumeClaim{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, createdPvc)
	assert.NoError(t, err)

	_, err = rc.Reconcile(t.Context())
	assert.NoError(t, err)

	updatedPvc := &corev1.PersistentVolumeClaim{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, updatedPvc)
	assert.NoError(t, err)

	assert.Equal(t, createdPvc.Generation, updatedPvc.Generation)
	assert.Zero(t, createdPvc.Spec.Resources.Requests.Storage().Cmp(*updatedPvc.Spec.Resources.Requests.Storage()))
}

func TestStorageSizeIncrease(t *testing.T) {
	storageClass := createStorageClass(t)
	defer k8sClient.Delete(t.Context(), storageClass)

	instance := defaultInstance.DeepCopy()
	instance.Spec.DataVolumeClaimSpec.StorageClassName = &[]string{"default-sc"}[0]
	rc := &reconciler.PVCReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	err := k8sClient.Create(t.Context(), instance)
	assert.NoError(t, err)

	defer cleanupPvcResources(t, instance)

	rc.Reconcile(t.Context())

	t.Log("Setting the PVC to bound")
	pvc := &corev1.PersistentVolumeClaim{}
	assert.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc))
	pvc.Status.Phase = corev1.ClaimBound
	assert.NoError(t, k8sClient.Status().Update(t.Context(), pvc))

	t.Log("Updating the storage size")
	instance.Spec.DataVolumeClaimSpec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("20Gi")
	assert.NoError(t, k8sClient.Update(t.Context(), instance))

	t.Log("Reconciling the updated instance")
	_, err = rc.Reconcile(t.Context())
	assert.NoError(t, err)

	pvc = &corev1.PersistentVolumeClaim{}
	assert.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc))
	assert.Zero(t, pvc.Spec.Resources.Requests.Storage().Cmp(*instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage()))
}

func TestStorageSizeDecrease(t *testing.T) {
	instance := defaultInstance.DeepCopy()
	rc := &reconciler.PVCReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	err := k8sClient.Create(t.Context(), instance)
	assert.NoError(t, err)

	defer cleanupPvcResources(t, instance)

	t.Log("Reconciling the setup phase")
	_, err = rc.Reconcile(t.Context())
	assert.NoError(t, err)

	t.Log("Updating the storage size")
	instance.Spec.DataVolumeClaimSpec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("5Gi")

	err = k8sClient.Update(t.Context(), instance)
	assert.NoError(t, err)

	t.Log("Reconciling the updated instance")
	_, err = rc.Reconcile(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "volume size decreased, not supported")
}

func createStorageClass(t *testing.T) *storagev1.StorageClass {
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-sc",
		},
		Parameters:  map[string]string{},
		Provisioner: "k8s.io/dummy-test",
		ReclaimPolicy: &[]corev1.PersistentVolumeReclaimPolicy{
			corev1.PersistentVolumeReclaimRetain,
		}[0],
		VolumeBindingMode: &[]storagev1.VolumeBindingMode{
			storagev1.VolumeBindingWaitForFirstConsumer,
		}[0],
		AllowVolumeExpansion: &[]bool{true}[0],
	}

	assert.NoError(t, k8sClient.Create(t.Context(), storageClass))
	return storageClass
}

func cleanupPvcResources(t *testing.T, instance *v1alpha1.LavinMQ) {
	err := k8sClient.Delete(t.Context(), instance)
	if err != nil {
		t.Fatalf("Failed to delete instance: %v", err)
	}
	for i := range int(instance.Spec.Replicas) {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("data-%s-%d", instance.Name, i),
				Namespace: instance.Namespace,
			},
		}

		err = k8sClient.Get(t.Context(), types.NamespacedName{Name: fmt.Sprintf("data-%s-%d", instance.Name, i), Namespace: instance.Namespace}, pvc)
		assert.NoErrorf(t, err, "Failed to get PVC")
		pvc.Finalizers = nil
		err = k8sClient.Update(t.Context(), pvc)
		assert.NoErrorf(t, err, "Failed to update PVC")

		err = k8sClient.Delete(t.Context(), pvc)
		if err != nil {
			// Only fail if error is not "not found"
			if !apierrors.IsNotFound(err) {
				t.Fatalf("Failed to delete PVC: %v", err)
			}
		}
	}
}
