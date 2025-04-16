package reconciler_test

import (
	"context"
	"fmt"
	"lavinmq-operator/api/v1alpha1"
	"lavinmq-operator/internal/reconciler"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("PVCReconciler", func() {
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
				Replicas: 1,
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
		for i := range int(instance.Spec.Replicas) {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("data-%s-%d", instance.Name, i),
					Namespace: instance.Namespace,
				},
			}

			k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("data-%s-%d", instance.Name, i), Namespace: instance.Namespace}, pvc)
			By("Removing PVC finalizer to allow deletion")
			pvc.Finalizers = nil
			err := k8sClient.Update(context.Background(), pvc)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Delete(context.Background(), pvc)
			if err != nil {
				// Only fail if error is not "not found"
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}
		}
	})

	When("using default values", func() {
		It("should create a PVC with the correct template", func() {
			rc.Reconcile(context.Background())

			pvc := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc)).To(Succeed())
			Expect(pvc).NotTo(BeNil())

			Expect(pvc.Name).To(Equal(fmt.Sprintf("data-%s-0", instance.Name)))
			Expect(pvc.Namespace).To(Equal(instance.Namespace))
			Expect(pvc.Spec.AccessModes).To(Equal(instance.Spec.DataVolumeClaimSpec.AccessModes))
			Expect(pvc.Spec.Resources.Requests.Storage().Cmp(*instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage())).To(Equal(0))
		})
	})

	When("there are no changes", func() {
		It("should not make any changes to the PVC", func() {
			By("Reconciling the instance setup")
			_, err := rc.Reconcile(context.Background())
			Expect(err).NotTo(HaveOccurred())

			By("Checking the PVC")
			pvc := &corev1.PersistentVolumeClaim{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc)
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc.Spec.Resources.Requests.Storage().Cmp(*instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage())).To(Equal(0))
		})
	})

	When("storage size increases", func() {
		var storageClass *storagev1.StorageClass

		BeforeEach(func() {
			storageClass = createStorageClass()
			instance.Spec.DataVolumeClaimSpec.StorageClassName = &[]string{"default-sc"}[0]
			err := k8sClient.Update(context.Background(), instance)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(context.Background(), storageClass)).To(Succeed())
		})

		It("should increase the PVC size", func() {
			By("Reconciling the instance setup")
			_, err := rc.Reconcile(context.Background())
			Expect(err).NotTo(HaveOccurred())

			By("Setting the PVC to bound")
			pvc := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("data-%s-0", instance.Name), Namespace: instance.Namespace}, pvc)).To(Succeed())
			pvc.Status.Phase = corev1.ClaimBound
			Expect(k8sClient.Status().Update(context.Background(), pvc)).To(Succeed())

			By("Updating the storage size")
			instance.Spec.DataVolumeClaimSpec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("20Gi")
			err = k8sClient.Update(context.Background(), instance)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the updated instance")
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
		It("should not allow the change and return an error", func() {
			By("Reconciling the setup phase")
			_, err := rc.Reconcile(context.Background())
			Expect(err).NotTo(HaveOccurred())

			By("Updating the storage size")
			instance.Spec.DataVolumeClaimSpec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("5Gi")

			err = k8sClient.Update(context.Background(), instance)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the updated instance")
			_, err = rc.Reconcile(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("volume size decreased, not supported"))
		})
	})
})

func createStorageClass() *storagev1.StorageClass {
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

	Expect(k8sClient.Create(context.Background(), storageClass)).To(Succeed())
	return storageClass
}
