package builder

import (
	"fmt"
	"lavinmq-operator/internal/controller/utils"

	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PVCBuilder struct {
	*ResourceBuilder
}

func (builder *ResourceBuilder) PVCBuilder() *PVCBuilder {
	return &PVCBuilder{
		ResourceBuilder: builder,
	}
}

func (b *PVCBuilder) Name() string {
	return "pvc"
}

func (b *PVCBuilder) NewObject() client.Object {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.Instance.Name,
			Namespace: b.Instance.Namespace,
			Labels:    utils.LabelsForLavinMQ(b.Instance),
		},
	}
}

func (b *PVCBuilder) Build() (client.Object, error) {
	pvc := b.NewObject().(*corev1.PersistentVolumeClaim)

	pvc.Spec = b.Instance.Spec.DataVolumeClaimSpec

	return pvc, nil
}

func (b *PVCBuilder) Diff(old, new client.Object) (client.Object, bool, error) {
	logger := b.Logger
	oldPVC := old.(*corev1.PersistentVolumeClaim)
	newPVC := new.(*corev1.PersistentVolumeClaim)
	diff := false

	// Check if storage class changed
	if oldPVC.Spec.StorageClassName != nil && newPVC.Spec.StorageClassName != nil {
		if *oldPVC.Spec.StorageClassName != *newPVC.Spec.StorageClassName {
			return newPVC, false, fmt.Errorf("storage class change not supported")
		}
	}

	// Check if access modes changed
	if !reflect.DeepEqual(oldPVC.Spec.AccessModes, newPVC.Spec.AccessModes) {
		return newPVC, false, fmt.Errorf("access modes change not supported")
	}

	// Handle storage size changes
	sizeComp := oldPVC.Spec.Resources.Requests.Storage().Cmp(*newPVC.Spec.Resources.Requests.Storage())
	switch sizeComp {
	case -1:
		logger.Info("Volume size changed, increasing",
			"old", oldPVC.Spec.Resources.Requests.Storage(),
			"new", newPVC.Spec.Resources.Requests.Storage())
		oldPVC.Spec.Resources.Requests = newPVC.Spec.Resources.Requests
		diff = true
	case 1:
		logger.Info("Volume size decreased, not supported")
		return newPVC, false, fmt.Errorf("volume size decreased, not supported")
	}

	// Update status if needed
	if oldPVC.Status.Phase != newPVC.Status.Phase {
		logger.Info("PVC status changed",
			"old", oldPVC.Status.Phase,
			"new", newPVC.Status.Phase)
		oldPVC.Status = newPVC.Status
	}

	return oldPVC, diff, nil
}
