package builder

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	lavinmqv1alpha1 "lavinmq-operator/api/v1alpha1"
)

var _ = Describe("StatefulSetBuilder", func() {
	var (
		instance       *lavinmqv1alpha1.LavinMQ
		reconciler     *StatefulSetReconciler // System Under Test
		namespacedName = types.NamespacedName{
			Name:      "test-lavinmq",
			Namespace: "default",
		}
	)

	BeforeEach(func() {
		instance = &lavinmqv1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-lavinmq",
				Namespace: "test-ns",
			},
			Spec: lavinmqv1alpha1.LavinMQSpec{
				Replicas: 1,
				Image:    "test-image:latest",
				DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{ // Default spec
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		}

		reconciler = &StatefulSetReconciler{
			ResourceBuilder: &ResourceBuilder{
				Instance: instance,
				Scheme:   scheme.Scheme,
				Client:   k8sClient,
			},
		}

		err := k8sClient.Create(context.Background(), instance)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Reconcile", func() {
		Context("When changing the image of the lavinmq cluster", func() {
			It("Should update the image of the lavinmq cluster", func() {
				instance.Spec.Image = "test-image:latest2"
				Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())

				_, err := reconciler.Reconcile(context.Background())
				Expect(err).NotTo(HaveOccurred())

				sts := &appsv1.StatefulSet{}
				err = k8sClient.Get(context.Background(), namespacedName, sts)
				Expect(err).NotTo(HaveOccurred())
				Expect(sts.Spec.Template.Spec.Containers[0].Image).To(Equal("test-image:latest2"))
			})
		})
	})
})
