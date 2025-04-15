package reconciler_test

import (
	"context"
	"fmt"
	"lavinmq-operator/api/v1alpha1"
	"lavinmq-operator/internal/reconciler"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("PVCBuilder", func() {
	var (
		instance       *v1alpha1.LavinMQ
		rc             *reconciler.PVCReconciler
		namespacedName = types.NamespacedName{
			Name:      "test-lavinmq",
			Namespace: "default",
		}
	)

	BeforeEach(func() {
		instance = &v1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: "default",
			},
			Spec: v1alpha1.LavinMQSpec{
				DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			},
		}
		rc = &reconciler.PVCReconciler{
			ResourceReconciler: &reconciler.ResourceReconciler{
				Instance: instance,
				Scheme:   scheme.Scheme,
				Client:   k8sClient,
				Logger:   log.FromContext(context.Background()),
			},
		}

		err := k8sClient.Create(context.Background(), instance)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), instance)).To(Succeed())
		for i := 0; i < int(instance.Spec.Replicas); i++ {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("data-%s-%d", instance.Name, i),
					Namespace: instance.Namespace,
				},
			}
			err := k8sClient.Delete(context.Background(), pvc)
			if err != nil {
				// Only fail if error is not "not found"
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}
		}
	})

	Describe("Build", func() {
		It("should create a PVC with the correct template", func() {
			err := k8sClient.Update(context.Background(), instance)
			Expect(err).NotTo(HaveOccurred())

			rc.Reconcile(context.Background())

			pvc := &corev1.PersistentVolumeClaim{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc)
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc).NotTo(BeNil())

			Expect(pvc.Name).To(Equal(fmt.Sprintf("data-%s-0", instance.Name)))
			Expect(pvc.Namespace).To(Equal(instance.Namespace))
			Expect(pvc.Spec.AccessModes).To(Equal(instance.Spec.DataVolumeClaimSpec.AccessModes))
			Expect(pvc.Spec.Resources.Requests.Storage().Cmp(*instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage())).To(Equal(0))
		})
	})

	Describe("Diff", func() {
		When("there are no changes", func() {
			It("should return no diff and no error", func() {
				err := k8sClient.Update(context.Background(), instance)
				Expect(err).NotTo(HaveOccurred())

				_, err = rc.Reconcile(context.Background())
				Expect(err).NotTo(HaveOccurred())
				pvc := &corev1.PersistentVolumeClaim{}
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc)
				Expect(err).NotTo(HaveOccurred())
				Expect(pvc.Spec.Resources.Requests.Storage().Cmp(*instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage())).To(Equal(0))
			})
		})

		When("storage size changes", func() {
			When("storage size increases", func() {
				It("should allow the change and return a diff", func() {
					Skip("skipping for now while expanding disks is not supported in test env")
					_, err := rc.Reconcile(context.Background())
					Expect(err).NotTo(HaveOccurred())
					instance.Spec.DataVolumeClaimSpec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("20Gi")

					pvc := &corev1.PersistentVolumeClaim{}
					err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc)

					fmt.Println(pvc)

					err = k8sClient.Update(context.Background(), instance)
					Expect(err).NotTo(HaveOccurred())

					_, err = rc.Reconcile(context.Background())
					Expect(err).NotTo(HaveOccurred())

					Expect(err).NotTo(HaveOccurred())

					pvc = &corev1.PersistentVolumeClaim{}
					err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc)
					Expect(err).NotTo(HaveOccurred())
					Expect(pvc.Spec.Resources.Requests.Storage().Cmp(*instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage())).To(Equal(0))
				})
			})

			When("storage size decreases", func() {
				It("should return an error", func() {
					instance.Spec.DataVolumeClaimSpec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("5Gi")

					err := k8sClient.Update(context.Background(), instance)
					Expect(err).NotTo(HaveOccurred())

					_, err = rc.Reconcile(context.Background())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("volume size decreased, not supported"))
				})
			})
		})
	})
})
