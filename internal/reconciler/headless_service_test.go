package reconciler_test

import (
	"context"
	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"
	"lavinmq-operator/internal/reconciler"
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
		instance *cloudamqpcomv1alpha1.LavinMQ
		rc       *reconciler.HeadlessServiceReconciler
	)

	BeforeEach(func() {
		instance = &cloudamqpcomv1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}

		rc = &reconciler.HeadlessServiceReconciler{
			ResourceReconciler: &reconciler.ResourceReconciler{
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
			rc.Reconcile(context.Background())

			service := &corev1.Service{}
			Expect(k8sClient.Get(context.Background(), namespacedName, service)).To(Succeed())
			Expect(service.Name).To(Equal(namespacedName.Name))
			Expect(service.Spec.ClusterIP).To(Equal("None"))
			Expect(service.Spec.Ports).To(HaveLen(3)) // amqp, http, and mqtt
		})
	})

	Context("When providing custom ports", func() {
		BeforeEach(func() {
			instance.Spec.Config.Amqp.Port = 1111
			instance.Spec.Config.Mgmt.Port = 2222
			instance.Spec.Config.Amqp.TlsPort = 3333
			instance.Spec.Config.Mgmt.TlsPort = 4444
			instance.Spec.Config.Mqtt.Port = 5555
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
		})

		It("Should create service with all specified ports", func() {
			rc.Reconcile(context.Background())

			service := &corev1.Service{}
			err := k8sClient.Get(context.Background(), namespacedName, service)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.Spec.Ports).To(HaveLen(5))
			i := slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Name == "amqp"
			})
			Expect(service.Spec.Ports[i].Port).To(Equal(int32(1111)))
			i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Name == "http"
			})
			Expect(service.Spec.Ports[i].Port).To(Equal(int32(2222)))
			i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Name == "amqps"
			})
			Expect(service.Spec.Ports[i].Port).To(Equal(int32(3333)))
			i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Name == "https"
			})
			Expect(service.Spec.Ports[i].Port).To(Equal(int32(4444)))
			i = slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Name == "mqtt"
			})
			Expect(service.Spec.Ports[i].Port).To(Equal(int32(5555)))
		})
	})

	Context("When clustering is enabled", func() {
		BeforeEach(func() {
			instance.Spec.EtcdEndpoints = []string{"etcd-0:2379"}
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
		})

		It("Should include clustering port", func() {
			rc.Reconcile(context.Background())

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
			instance.Spec.Config.Amqp.Port = 5672
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
			rc.Reconcile(context.Background())
		})

		It("Should update ports when they change", func() {
			instance.Spec.Config.Amqp.Port = 1111
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
			rc.Reconcile(context.Background())

			service := &corev1.Service{}
			Expect(k8sClient.Get(context.Background(), namespacedName, service)).To(Succeed())
			i := slices.IndexFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Name == "amqp"
			})
			Expect(service.Spec.Ports[i].Port).To(Equal(int32(1111)))
		})
	})
})
