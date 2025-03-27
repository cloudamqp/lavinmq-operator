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
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"
)

// EtcdReconciler reconciles a Etcd object
type EtcdReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cloudamqp.com.cloudamqp.com,resources=etcds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudamqp.com.cloudamqp.com,resources=etcds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudamqp.com.cloudamqp.com,resources=etcds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Etcd object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *EtcdReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	instance := &cloudamqpcomv1alpha1.Etcd{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Etcd not found, either deleted or never created")
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to get Etcd")
		return ctrl.Result{}, err
	}

	logger.Info("Etcd found", "name", instance.Name)

	found := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("StatefulSet not found, creating")
			statefulset, err := r.createStatefulSet(ctx, instance)
			if err != nil {
				logger.Error(err, "Failed to create StatefulSet for Etcd")
				return ctrl.Result{}, err
			}

			logger.Info("Creating StatefulSet for Etcd", "name", statefulset.Name)

			if err := r.Create(ctx, statefulset); err != nil {
				logger.Error(err, "Failed to create StatefulSet for Etcd",
					"Deployment.Namespace", statefulset.Namespace,
					"Deployment.Name", statefulset.Name)
				return ctrl.Result{}, err
			}

			logger.Info("Created StatefulSet for Etcd", "name", statefulset.Name)

			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
	}

	foundService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundService)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Service not found, creating")
			service, err := r.createService(ctx, instance)
			if err != nil {
				logger.Error(err, "Failed to create Service for Etcd")
				return ctrl.Result{}, err
			}

			logger.Info("Creating Service for Etcd", "name", service.Name)

			if err := r.Create(ctx, service); err != nil {
				logger.Error(err, "Failed to create Service for Etcd",
					"Service.Namespace", service.Namespace,
					"Service.Name", service.Name)
				return ctrl.Result{}, err
			}

			logger.Info("Created Service for Etcd", "name", service.Name)

			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudamqpcomv1alpha1.Etcd{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *EtcdReconciler) createStatefulSet(ctx context.Context, instance *cloudamqpcomv1alpha1.Etcd) (*appsv1.StatefulSet, error) {
	labels := labelsForEtcd(instance)
	replicas := instance.Spec.Replicas
	ports := instance.Spec.Ports
	volume := instance.Spec.DataVolumeClaimSpec
	volumeName := instance.Name + "-data"

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
							Name:  "etcd",
							Image: image,
							Ports: ports,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: "/var/lib/etcd",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "ALLOW_NONE_AUTHENTICATION",
									Value: "yes",
								},
								{
									Name:  "ETCD_LISTEN_CLIENT_URLS",
									Value: "http://0.0.0.0:2379",
								},
								{
									Name:  "ETCD_LISTEN_PEER_URLS",
									Value: "http://0.0.0.0:2380",
								},
								{
									Name:  "ETCD_ADVERTISE_CLIENT_URLS",
									Value: "http://etcd-sample:2379",
								},
								{
									Name:  "ETCD_INITIAL_ADVERTISE_PEER_URLS",
									Value: "http://etcd-sample:2380",
								},
								{
									Name:  "ETCD_INITIAL_CLUSTER",
									Value: "etcd-sample=http://etcd-sample:2380",
								},
								{
									Name:  "ETCD_DATA_DIR",
									Value: "/var/lib/etcd",
								},
								{
									Name:  "ETCD_NAME",
									Value: instance.Name,
								},
							},
						},
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

func (r *EtcdReconciler) createService(ctx context.Context, instance *cloudamqpcomv1alpha1.Etcd) (*corev1.Service, error) {
	labels := labelsForEtcd(instance)
	ports := instance.Spec.Ports
	servicePorts := []corev1.ServicePort{}
	for _, port := range ports {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       port.Name,
			Port:       port.ContainerPort,
			TargetPort: intstr.FromInt(int(port.ContainerPort)),
			Protocol:   "TCP",
		})
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector:  labels,
			ClusterIP: "None",
			Ports:     servicePorts,
		},
	}

	// Setting owner reference
	if err := ctrl.SetControllerReference(instance, service, r.Scheme); err != nil {
		return nil, err
	}

	return service, nil
}

func labelsForEtcd(instance *cloudamqpcomv1alpha1.Etcd) map[string]string {
	image := instance.Spec.Image
	version := strings.Split(image, ":")[1]

	labels := map[string]string{
		"app.kubernetes.io/name":       "lavinmq-operator",
		"app.kubernetes.io/managed-by": "EtcdController",
		"app.kubernetes.io/version":    version,
	}

	// Append instance labels
	for k, v := range instance.Labels {
		labels[k] = v
	}

	return labels
}
