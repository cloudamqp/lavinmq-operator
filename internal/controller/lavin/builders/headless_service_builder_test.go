package builder

import (
	"context"
	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("HeadlessServiceReconciler", func() {
	var namespacedName = types.NamespacedName{
		Name:      "test-resource",
		Namespace: "default",
	}
	var (
		instance   *cloudamqpcomv1alpha1.LavinMQ
		reconciler *HeadlessServiceReconciler
	)

	BeforeEach(func() {
		instance = &cloudamqpcomv1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}

		reconciler = &HeadlessServiceReconciler{
			ResourceBuilder: &ResourceBuilder{
				Instance: instance,
				Scheme:   scheme.Scheme,
				Client:   k8sClient,
				Logger:   log.FromContext(context.Background()),
			},
		}

		Expect(k8sClient.Create(context.Background(), instance)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), instance)).To(Succeed())
	})

	Context("When building a default Service", func() {
		It("Should return a headless service with default ports", func() {
			reconciler.Reconcile(context.Background())

			service := &corev1.Service{}
			Expect(k8sClient.Get(context.Background(), namespacedName, service)).To(Succeed())
			Expect(service.Name).To(Equal(namespacedName.Name))
			Expect(service.Spec.ClusterIP).To(Equal("None"))
			Expect(service.Spec.Ports).To(HaveLen(3)) // amqp, http, and mqtt
		})
	})

	Context("When providing custom ports", func() {
		BeforeEach(func() {
			instance.Spec.Ports = []corev1.ContainerPort{
				{
					Name:          "amqp",
					ContainerPort: 1111,
				},
				{
					Name:          "http",
					ContainerPort: 2222,
				},
				{
					Name:          "amqps",
					ContainerPort: 3333,
				},
				{
					Name:          "https",
					ContainerPort: 4444,
				},
			}

			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
		})

		It("Should create service with all specified ports", func() {
			reconciler.Reconcile(context.Background())

			service := &corev1.Service{}
			err := k8sClient.Get(context.Background(), namespacedName, service)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.Spec.Ports).To(HaveLen(4))
			Expect(service.Spec.Ports[0].Name).To(Equal("amqp"))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(1111)))
			Expect(service.Spec.Ports[1].Name).To(Equal("http"))
			Expect(service.Spec.Ports[1].Port).To(Equal(int32(2222)))
			Expect(service.Spec.Ports[2].Name).To(Equal("amqps"))
			Expect(service.Spec.Ports[2].Port).To(Equal(int32(3333)))
			Expect(service.Spec.Ports[3].Name).To(Equal("https"))
			Expect(service.Spec.Ports[3].Port).To(Equal(int32(4444)))
		})
	})

	Context("When clustering is enabled", func() {
		BeforeEach(func() {
			instance.Spec.EtcdEndpoints = []string{"etcd-0:2379"}
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
		})

		It("Should include clustering port", func() {
			reconciler.Reconcile(context.Background())

			service := &corev1.Service{}
			err := k8sClient.Get(context.Background(), namespacedName, service)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.Spec.Ports).To(HaveLen(4)) // amqp, http, mqtt and clustering
			idx := slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Name == "clustering"
			})
			Expect(idx).NotTo(Equal(-1))
			Expect(service.Spec.Ports[idx].Port).To(Equal(int32(5679)))
		})
	})

	Context("When updating fields", func() {
		BeforeEach(func() {
			instance.Spec.Ports = []corev1.ContainerPort{
				{
					Name:          "amqp",
					ContainerPort: 5672,
				},
			}
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
			reconciler.Reconcile(context.Background())
		})

		It("Should update ports when they change", func() {
			instance.Spec.Ports = []corev1.ContainerPort{
				{
					Name:          "amqp",
					ContainerPort: 1111,
				},
			}
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
			reconciler.Reconcile(context.Background())

			service := &corev1.Service{}
			Expect(k8sClient.Get(context.Background(), namespacedName, service)).To(Succeed())
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(1111)))
		})

		It("Should not update ports when they are the same", func() {
			instance.Spec.Ports = []corev1.ContainerPort{
				{
					Name:          "amqp",
					ContainerPort: 5672,
				},
			}
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
			reconciler.Reconcile(context.Background())

			service := &corev1.Service{}
			Expect(k8sClient.Get(context.Background(), namespacedName, service)).To(Succeed())
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(5672)))
		})
	})
})
