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
	"reflect"
	"strings"

	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"
	builder "lavinmq-operator/internal/controller/lavin/builders"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

// +kubebuilder:rbac:groups=cloudamqp.com.cloudamqp.com,resources=lavinmqs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudamqp.com.cloudamqp.com,resources=lavinmqs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudamqp.com.cloudamqp.com,resources=lavinmqs/finalizers,verbs=update

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

	// TODO(user): your logic here
	fmt.Printf("Reconciling LavinMQ %s\n", req.NamespacedName)

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

	sts := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sts)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("StatefulSet not found, creating")
			sts, err = r.createStatefulSet(ctx, instance)
			if err != nil {
				logger.Error(err, "Failed to create StatefulSet for LavinMQ")

				meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
					Type:    typeAvailableLavinMQ, // TODO: Is this correct?
					Status:  metav1.ConditionFalse,
					Reason:  "Reconciling",
					Message: fmt.Sprintf("Failed to create StatefulSet for LavinMQ: %s : %s", instance.Name, err),
				})

				if err := r.Status().Update(ctx, instance); err != nil {
					logger.Error(err, "Failed to update LavinMQ status")
					return ctrl.Result{}, err
				}

				return ctrl.Result{}, err
			}

			logger.Info("Creating StatefulSet for LavinMQ", "name", sts.Name)

			if err := r.Create(ctx, sts); err != nil {
				logger.Error(err, "Failed to create StatefulSet for LavinMQ",
					"Deployment.Namespace", sts.Namespace,
					"Deployment.Name", sts.Name)
				return ctrl.Result{}, err
			}

			logger.Info("Created StatefulSet for LavinMQ", "name", sts.Name)

			builder := builder.ServiceConfigBuilder{
				Instance: instance,
				Scheme:   r.Scheme,
			}

			configMap, err := builder.Build()
			if err != nil {
				logger.Error(err, "Failed to create ConfigMap for LavinMQ")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, configMap); err != nil {
				logger.Error(err, "Failed to create ConfigMap for LavinMQ")
				return ctrl.Result{}, err
			}
		}
	}

	for index, container := range sts.Spec.Template.Spec.Containers {
		if container.Name == "lavinmq" {
			if reflect.DeepEqual(instance.Spec.Ports, container.Ports) {
				logger.Info("Ports are the same, skipping")
				break
			}
			sts.Spec.Template.Spec.Containers[index].Ports = instance.Spec.Ports
			logger.Info("Ports are different, updating")
			builder := builder.ServiceConfigBuilder{
				Instance: instance,
				Scheme:   r.Scheme,
			}

			configMap, err := builder.Build()
			if err != nil {
				logger.Error(err, "Failed to create ConfigMap for LavinMQ")
				return ctrl.Result{}, err
			}

			if err := r.Update(ctx, configMap); err != nil {
				logger.Error(err, "Failed to update ConfigMap for LavinMQ")
				return ctrl.Result{}, err
			}

			logger.Info("Updated ConfigMap for LavinMQ", "name", configMap.Name)
		}
	}

	for index, container := range sts.Spec.Template.Spec.Containers {
		if container.Name == "lavinmq" {
			if reflect.DeepEqual(instance.Spec.Image, container.Image) {
				logger.Info("Image is the same, skipping")
				break
			}

			sts.Spec.Template.Spec.Containers[index].Image = instance.Spec.Image
			logger.Info("Image is different, updating")
		}
	}

	logger.Info("Reapplying stuff")
	if err := r.Update(ctx, sts); err != nil {
		logger.Error(err, "Failed to update StatefulSet for LavinMQ")
		return ctrl.Result{}, err
	}

	logger.Info("Updated StatefulSet for LavinMQ after port change", "name", sts.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LavinMQReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudamqpcomv1alpha1.LavinMQ{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		// May need deployment idk
		//Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *LavinMQReconciler) createStatefulSet(ctx context.Context, instance *cloudamqpcomv1alpha1.LavinMQ) (*appsv1.StatefulSet, error) {
	labels := labelsForLavinMQ(instance)
	replicas := instance.Spec.Replicas
	ports := instance.Spec.Ports
	volume := instance.Spec.DataVolumeClaimSpec
	volumeName := instance.Name + "-data"
	configVolumeName := fmt.Sprintf("%s-config", instance.Name)

	secretVolume, secretVolumeMount := r.referenceSecret(instance)

	image := instance.Spec.Image
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: instance.Name,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "lavinmq",
							Image: image,
							Ports: ports,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: "/var/lib/lavinmq",
								},
								{
									Name:      configVolumeName,
									MountPath: "/etc/lavinmq",
									ReadOnly:  true,
								},
								*secretVolumeMount,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: configVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: configVolumeName},
								},
							},
						},
						*secretVolume,
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: volumeName,
					},
					Spec: volume,
				},
			},
		},
	}

	// Setting owner reference
	if err := ctrl.SetControllerReference(instance, statefulset, r.Scheme); err != nil {
		return nil, err
	}

	return statefulset, nil
}

func (r *LavinMQReconciler) referenceSecret(instance *cloudamqpcomv1alpha1.LavinMQ) (*corev1.Volume, *corev1.VolumeMount) {
	if instance.Spec.Secrets == nil {
		return nil, nil
	}

	volume := &corev1.Volume{
		Name: "tls",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: instance.Spec.Secrets[0].Name,
			},
		},
	}

	volumeMount := &corev1.VolumeMount{
		Name:      "tls",
		MountPath: "/etc/lavinmq/tls",
		ReadOnly:  true,
	}

	return volume, volumeMount
}

func labelsForLavinMQ(instance *cloudamqpcomv1alpha1.LavinMQ) map[string]string {
	image := instance.Spec.Image
	version := strings.Split(image, ":")[1]

	return map[string]string{
		"app.kubernetes.io/name":       "lavinmq-operator",
		"app.kubernetes.io/managed-by": "LavinMQController",
		"app.kubernetes.io/version":    version,
	}
}
