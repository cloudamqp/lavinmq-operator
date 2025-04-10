package builder

import (
	"context"
	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceBuilder struct {
	Instance *cloudamqpcomv1alpha1.LavinMQ
	Scheme   *runtime.Scheme
	Logger   logr.Logger
	Client   client.Client
}

func (reconciler *ResourceBuilder) Reconcilers() []Reconciler {
	return []Reconciler{
		reconciler.ConfigReconciler(),
		reconciler.HeadlessServiceReconciler(),
		reconciler.PVCReconciler(),
		// reconciler.StatefulSetReconciler(client),
	}
}

type Reconciler interface {
	// TODO: Fix config restart context.
	Reconcile(ctx context.Context) (ctrl.Result, error)
}

func (reconciler *ResourceBuilder) GetItem(ctx context.Context, obj client.Object) error {
	err := reconciler.Client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
	if err != nil {
		return err
	}

	return nil
}

func (reconciler *ResourceBuilder) CreateItem(ctx context.Context, obj client.Object) error {

	reconciler.Logger.Info("Creating item", "name", obj.GetName(), "namespace", obj.GetNamespace())
	// Set owner reference
	if err := ctrl.SetControllerReference(reconciler.Instance, obj, reconciler.Scheme); err != nil {
		reconciler.Logger.Error(err, "Failed to set controller reference", "name", obj.GetName())
		return err
	}

	if err := reconciler.Client.Create(ctx, obj); err != nil {
		reconciler.Logger.Error(err, "Failed to create resource", "name", obj.GetName())
		return err
	}

	return nil
}
