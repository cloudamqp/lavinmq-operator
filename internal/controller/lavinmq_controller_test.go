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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"
)

var _ = Describe("LavinMQ Controller", func() {
	var (
		resourceName       = "test-resource"
		ctx                = context.Background()
		lavinmq            *cloudamqpcomv1alpha1.LavinMQ
		typeNamespacedName types.NamespacedName
		reconciler         *LavinMQReconciler
	)

	BeforeEach(func() {
		typeNamespacedName = types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		// Base resource configuration
		lavinmq = &cloudamqpcomv1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
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

		reconciler = &LavinMQReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	AfterEach(func() {
		// Clean up StatefulSet
		sts := &appsv1.StatefulSet{}
		err := k8sClient.Get(ctx, typeNamespacedName, sts)
		if err == nil {
			Expect(k8sClient.Delete(ctx, sts)).To(Succeed())
		}

		// Clean up ConfigMap
		configMap := &corev1.ConfigMap{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-config", resourceName), Namespace: "default"}, configMap)
		if err == nil {
			Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
		}

		// Clean up LavinMQ
		resource := &cloudamqpcomv1alpha1.LavinMQ{}
		err = k8sClient.Get(ctx, typeNamespacedName, resource)
		if err == nil {
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		}
	})

	Context("When a instance of LavinMQ doesn't exist", func() {
		It("Should not return a error", func() {
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When a instance of LavinMQ exists", func() {
		Context("When creating a default lavinmq resource", func() {
			BeforeEach(func() {
				By("creating the custom resource for the Kind LavinMQ")
				Expect(k8sClient.Create(ctx, lavinmq)).To(Succeed())
			})

			It("should verify the default container ports", func() {
				resource := &cloudamqpcomv1alpha1.LavinMQ{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.Spec.Ports).To(Equal([]corev1.ContainerPort{
					{ContainerPort: 5672, Name: "amqp", Protocol: "TCP"},
					{ContainerPort: 15672, Name: "http", Protocol: "TCP"},
					{ContainerPort: 1883, Name: "mqtt", Protocol: "TCP"},
				}))
			})

			It("should verify the default image", func() {
				resource := &cloudamqpcomv1alpha1.LavinMQ{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.Spec.Image).To(Equal("cloudamqp/lavinmq:2.2.0"))
			})

			It("should verify the default replicas", func() {
				resource := &cloudamqpcomv1alpha1.LavinMQ{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.Spec.Replicas).To(Equal(int32(1)))
			})
		})

		Context("When creating a lavinmq cluster with custom ports", func() {
			BeforeEach(func() {
				lavinmq.Spec.Ports = []corev1.ContainerPort{
					{ContainerPort: 1337, Name: "amqp", Protocol: "TCP"},
				}
				Expect(k8sClient.Create(ctx, lavinmq)).To(Succeed())
			})

			It("Should respect provided container ports", func() {
				resource := &cloudamqpcomv1alpha1.LavinMQ{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.Spec.Ports).To(Equal([]corev1.ContainerPort{
					{ContainerPort: 1337, Name: "amqp", Protocol: "TCP"},
				}))
			})
		})

		Context("When creating a lavinmq cluster with custom image", func() {
			BeforeEach(func() {
				lavinmq.Spec.Image = "cloudamqp/lavinmq:2.3.0"
				Expect(k8sClient.Create(ctx, lavinmq)).To(Succeed())
			})

			It("Should respect provided image", func() {
				resource := &cloudamqpcomv1alpha1.LavinMQ{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.Spec.Image).To(Equal("cloudamqp/lavinmq:2.3.0"))
			})
		})

		Context("When creating a lavinmq cluster with custom image", func() {
			BeforeEach(func() {
				lavinmq.Spec.Replicas = 3
				Expect(k8sClient.Create(ctx, lavinmq)).To(Succeed())
			})

			It("Should respect provided replicas", func() {
				resource := &cloudamqpcomv1alpha1.LavinMQ{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.Spec.Replicas).To(Equal(int32(3)))
			})
		})

		Context("When updating the ports of the lavinmq cluster", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(ctx, lavinmq)).To(Succeed())
				result, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))
				lavinmq.Spec.Ports = []corev1.ContainerPort{
					{ContainerPort: 1337, Name: "amqp", Protocol: "TCP"},
				}
				Expect(k8sClient.Update(ctx, lavinmq)).To(Succeed())
			})

			It("Should update the ports of the lavinmq cluster", func() {
				By("reconciling the resource")
				result, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))
				resource := &appsv1.StatefulSet{}
				err = k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.Spec.Template.Spec.Containers[0].Ports).To(Equal([]corev1.ContainerPort{
					{ContainerPort: 1337, Name: "amqp", Protocol: "TCP"},
				}))
			})
		})

		Context("When updating the image of the lavinmq cluster", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(ctx, lavinmq)).To(Succeed())
				result, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))
				lavinmq.Spec.Image = "cloudamqp/lavinmq:2.3.0"
				Expect(k8sClient.Update(ctx, lavinmq)).To(Succeed())
			})

			It("Should update the image of the lavinmq cluster", func() {
				By("reconciling the resource")
				result, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))
				resource := &appsv1.StatefulSet{}
				err = k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.Spec.Template.Spec.Containers[0].Image).To(Equal("cloudamqp/lavinmq:2.3.0"))
			})
		})
	})
})
