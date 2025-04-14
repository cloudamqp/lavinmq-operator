package builder

import (
	"context"
	"fmt"
	"lavinmq-operator/internal/controller/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type PVCReconciler struct {
	*ResourceBuilder
}

func (reconciler *ResourceBuilder) PVCReconciler() *PVCReconciler {
	return &PVCReconciler{
		ResourceBuilder: reconciler,
	}
}

func (b *PVCReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	pvcs := b.newObjects()

	for _, pvc := range pvcs {
		err := b.GetItem(ctx, &pvc)
		if err != nil {
			if apierrors.IsNotFound(err) {
				b.CreateItem(ctx, &pvc)
				continue
			}

			return ctrl.Result{}, err
		}

		err = b.updateFields(ctx, &pvc)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = b.Client.Update(ctx, &pvc)
		if err != nil {
			b.Logger.Error(err, "Failed to update PVC")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (b *PVCReconciler) newObjects() []corev1.PersistentVolumeClaim {
	pvcs := []corev1.PersistentVolumeClaim{}

	for i := 0; i < int(b.Instance.Spec.Replicas); i++ {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("data-%s-%d", b.Instance.Name, i),
				Namespace: b.Instance.Namespace,
				Labels:    utils.LabelsForLavinMQ(b.Instance),
			},
			Spec: b.Instance.Spec.DataVolumeClaimSpec,
		}
		// Forcing ReadWriteOnce for volume access mode
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		pvcs = append(pvcs, *pvc)
	}

	return pvcs
}

func (b *PVCReconciler) updateFields(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	sizeComp := pvc.Spec.Resources.Requests.Storage().Cmp(*b.Instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage())

	switch sizeComp {
	case -1:
		b.Logger.Info("Volume size changed, increasing",
			"old", pvc.Spec.Resources.Requests.Storage(),
			"new", b.Instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage())
		storageClassName := *pvc.Spec.StorageClassName

		storageClass := &corev1.StorageClass{}
		err := b.Client.Get(ctx, types.NamespacedName{Name: *storageClassName}, storageClass)
		if err != nil {
			return err
		}

		oldPvc := pvc.DeepCopy()
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = b.Instance.Spec.DataVolumeClaimSpec.Resources.Requests[corev1.ResourceStorage]
		b.Logger.Info("Volume size changed", "old", oldPvc.Spec.Resources.Requests.Storage(), "new", pvc.Spec.Resources.Requests.Storage())
	case 1:
		b.Logger.Info("Volume size decreased, not supported")
		return fmt.Errorf("volume size decreased, not supported")
	}

	return nil
}

// Name returns the name of the PVC reconciler
func (b *PVCReconciler) Name() string {
	return "pvc"
}
