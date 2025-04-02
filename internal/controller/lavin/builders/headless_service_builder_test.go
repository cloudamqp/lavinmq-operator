package builder

import (
	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("HeadlessServiceBuilder", func() {

	var (
		objectMeta = metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		}
		builder HeadlessServiceBuilder
	)

	Context("When building a HSService without port or etcdendpoint spec", func() {
		BeforeEach(func() {
			instance := &cloudamqpcomv1alpha1.LavinMQ{
				ObjectMeta: objectMeta,
			}
			builder = HeadlessServiceBuilder{
				ResourceBuilder: &ResourceBuilder{
					Instance: instance,
					Scheme:   scheme.Scheme,
				},
			}
		})

		It("Should return a empty service", func() {
			obj, err := builder.Build()
			HeadlessService := obj.(*corev1.Service)
			Expect(err).NotTo(HaveOccurred())
			Expect(HeadlessService.Name).To(Equal("test-resource-service"))
			Expect(HeadlessService.Spec.Ports).To(HaveLen(0))
			Expect(HeadlessService.Spec.ClusterIP).To(Equal("None"))
		})
	})

	Context("When building a headlessService with port spec and etcd", func() {
		BeforeEach(func() {
			instance := &cloudamqpcomv1alpha1.LavinMQ{
				ObjectMeta: objectMeta,
				Spec: cloudamqpcomv1alpha1.LavinMQSpec{
					Ports: []corev1.ContainerPort{
						{
							Name:          "amqp",
							ContainerPort: 5672,
						},
						{
							Name:          "http",
							ContainerPort: 15672,
						},
					},
					EtcdEndpoints: []string{"localhost:2379"},
				},
			}
			builder = HeadlessServiceBuilder{
				ResourceBuilder: &ResourceBuilder{
					Instance: instance,
					Scheme:   scheme.Scheme,
				},
			}
		})

		It("Should return a service for including both clustering and specified ports", func() {
			obj, err := builder.Build()
			HeadlessService := obj.(*corev1.Service)
			Expect(err).NotTo(HaveOccurred())
			Expect(HeadlessService.Name).To(Equal("test-resource-service"))
			Expect(HeadlessService.Spec.ClusterIP).To(Equal("None"))
			Expect(HeadlessService.Spec.Ports).To(HaveLen(3))
		})
	})

	Context("When building a headlessService with port spec", func() {
		BeforeEach(func() {
			instance := &cloudamqpcomv1alpha1.LavinMQ{
				ObjectMeta: objectMeta,
				Spec: cloudamqpcomv1alpha1.LavinMQSpec{
					Ports: []corev1.ContainerPort{
						{
							Name:          "amqp",
							ContainerPort: 5672,
						},
						{
							Name:          "http",
							ContainerPort: 15672,
						},
					},
				},
			}
			builder = HeadlessServiceBuilder{
				ResourceBuilder: &ResourceBuilder{
					Instance: instance,
					Scheme:   scheme.Scheme,
				},
			}
		})

		It("Should return a service with both ports includes", func() {
			obj, err := builder.Build()
			HeadlessService := obj.(*corev1.Service)
			Expect(err).NotTo(HaveOccurred())
			Expect(HeadlessService.Name).To(Equal("test-resource-service"))
			Expect(HeadlessService.Spec.ClusterIP).To(Equal("None"))
			Expect(HeadlessService.Spec.Ports).To(HaveLen(2))
		})
	})

	Context("When building a HeadlessService with etcdendpoint spec", func() {
		BeforeEach(func() {
			instance := &cloudamqpcomv1alpha1.LavinMQ{
				ObjectMeta: objectMeta,
				Spec: cloudamqpcomv1alpha1.LavinMQSpec{
					EtcdEndpoints: []string{"localhost:2379"},
				},
			}
			builder = HeadlessServiceBuilder{
				ResourceBuilder: &ResourceBuilder{
					Instance: instance,
					Scheme:   scheme.Scheme,
				},
			}
		})

		It("Should return a service for port 5679", func() {
			obj, err := builder.Build()
			HeadlessService := obj.(*corev1.Service)
			Expect(err).NotTo(HaveOccurred())
			Expect(HeadlessService.Name).To(Equal("test-resource-service"))
			Expect(HeadlessService.Spec.ClusterIP).To(Equal("None"))
			Expect(HeadlessService.Spec.Ports).To(HaveLen(1))
			Expect(HeadlessService.Spec.Ports[0].Port).To(Equal(int32(5679)))
			Expect(HeadlessService.Spec.Ports[0].Name).To(Equal("clustering"))
		})
	})

	Context("When diffing headless services", func() {
		var (
			oldService *corev1.Service
			newService *corev1.Service
		)

		BeforeEach(func() {
			oldService = &corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:       "amqp",
							Port:       5672,
							TargetPort: intstr.FromInt(5672),
							Protocol:   "TCP",
						},
					},
				},
			}
			newService = &corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:       "amqp",
							Port:       5672,
							TargetPort: intstr.FromInt(5672),
							Protocol:   "TCP",
						},
					},
				},
			}
		})

		It("should return no diff when ports are identical", func() {
			result, diff, err := builder.Diff(oldService, newService)
			Expect(err).NotTo(HaveOccurred())
			Expect(diff).To(BeFalse())
			Expect(result).To(Equal(oldService))
		})

		It("should return a diff when ports change", func() {
			newService.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "amqp",
					Port:       5672,
					TargetPort: intstr.FromInt(5672),
					Protocol:   "TCP",
				},
				{
					Name:       "http",
					Port:       15672,
					TargetPort: intstr.FromInt(15672),
					Protocol:   "TCP",
				},
			}

			result, diff, err := builder.Diff(oldService, newService)
			Expect(err).NotTo(HaveOccurred())
			Expect(diff).To(BeTrue())
			Expect(result).To(Equal(newService))
		})
	})
})
