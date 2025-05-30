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

	cloudamqpcomv1alpha1 "github.com/cloudamqp/lavinmq-operator/api/v1alpha1"
	"github.com/cloudamqp/lavinmq-operator/internal/reconciler"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Definitions to manage status conditions
const (
	// typeAvailableLavinMQ represents the status of the StatefulSet reconciliation
	typeAvailableLavinMQ = "Available"
	// typeDegradedLavinMQ represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	typeDegradedLavinMQ = "Degraded"
)

// LavinMQReconciler reconciles a LavinMQ object
type LavinMQReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=cloudamqp.com,resources=lavinmqs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudamqp.com,resources=lavinmqs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudamqp.com,resources=lavinmqs/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LavinMQ object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *LavinMQReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("Reconciling LavinMQ %s\n", req.NamespacedName))

	instance := &cloudamqpcomv1alpha1.LavinMQ{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("LavinMQ not found, either deleted or never created")
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to get LavinMQ")
		return ctrl.Result{}, err
	}

	logger.Info("LavinMQ found", "name", instance.Name)
	resourceReconciler := reconciler.ResourceReconciler{
		Instance: instance,
		Scheme:   r.Scheme,
		Logger:   logger,
		Client:   r.Client,
	}

	reconcilers := resourceReconciler.Reconcilers()

	for _, reconciler := range reconcilers {
		_, err := reconciler.Reconcile(ctx)
		if err != nil {
			logger.Error(err, "Failed to reconcile resource", "name", reconciler.Name())
			return ctrl.Result{}, err
		}
	}

	logger.Info("Updated resources for LavinMQ")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LavinMQReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudamqpcomv1alpha1.LavinMQ{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
