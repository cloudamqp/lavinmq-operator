package builder

import (
	"context"
	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ini "gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func verifyConfigMapEquality(configMap *corev1.ConfigMap, expectedConfig string) {
	conf, _ := ini.Load([]byte(configMap.Data["lavinmq.ini"]))
	expectedConf, _ := ini.Load([]byte(expectedConfig))

	for _, section := range conf.Sections() {
		for _, key := range section.Keys() {
			val := conf.Section(section.Name()).Key(key.Name()).Value()
			Expect(expectedConf.Section(section.Name()).Key(key.Name()).Value()).To(Equal(val))
		}
	}
}

var _ = Describe("ConfigBuilder", func() {
	var namespacedName = types.NamespacedName{
		Name:      "test-resource",
		Namespace: "default",
	}
	var (
		instance = &cloudamqpcomv1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}
		builder *ConfigReconciler
	)

	BeforeEach(func() {
		builder = &ConfigReconciler{
			ResourceBuilder: &ResourceBuilder{
				Instance: instance,
				Scheme:   scheme.Scheme,
				Client:   k8sClient,
			},
		}

		Expect(k8sClient.Create(context.Background(), instance)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), instance)).To(Succeed())
	})

	Context("When building a default ConfigMap", func() {
		var expectedConfig = `
			[main]
			log_level = info
			data_dir = /var/lib/lavinmq

			[mgmt]
			bind = 0.0.0.0

			[amqp]
			bind = 0.0.0.0
			heartbeat = 300

			[clustering]
			enabled = true
			bind = 0.0.0.0
			port = 5679
			advertised_uri = tcp://test-resource:5679
	`
		It("Should return a default ConfigMap", func() {
			builder.Reconcile(context.Background())

			configMap := &corev1.ConfigMap{}

			err := k8sClient.Get(context.Background(), namespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Name).To(Equal(namespacedName.Name))
			verifyConfigMapEquality(configMap, expectedConfig)
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

		expectedConfig := `
			[main]
			log_level = info
			data_dir = /var/lib/lavinmq

			[mgmt]
			bind = 0.0.0.0
			port = 2222
			tls_port = 4444

			[amqp]
			bind = 0.0.0.0
			heartbeat = 300
			port = 1111
			tls_port = 3333

			[clustering]
			enabled = true
			bind = 0.0.0.0
			port = 5679
			advertised_uri = tcp://test-resource:5679
		`

		It("Should setup ports in according section", func() {
			builder.Reconcile(context.Background())

			configMap := &corev1.ConfigMap{}
			err := k8sClient.Get(context.Background(), namespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Name).To(Equal(namespacedName.Name))
			verifyConfigMapEquality(configMap, expectedConfig)
		})

	})

	// I think this test is not needed anymore with the new "reconciler" approach where it updates the configmap directly.
	// 	Context("When diffing config maps", func() {
	// 		var (
	// 			oldConfigMap *corev1.ConfigMap
	// 		)

	// 		BeforeEach(func() {
	// 			oldConfigMap = &corev1.ConfigMap{
	// 				Data: map[string]string{
	// 					"lavinmq.ini": `
	// [main]
	// log_level = info
	// data_dir = /var/lib/lavinmq

	// [mgmt]
	// bind = 0.0.0.0

	// [amqp]
	// bind = 0.0.0.0
	// heartbeat = 300

	// [clustering]
	// enabled = true
	// bind = 0.0.0.0
	// port = 5679
	// advertised_uri = tcp://test-resource:5679`,
	// 				},
	// 			}
	// 		})

	// 		It("should return no diff when configs are identical", func() {
	// 			builder.Reconcile(context.Background())

	//			configMap := &corev1.ConfigMap{}
	//			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-resource-config", Namespace: "default"}, configMap)
	//			Expect(err).NotTo(HaveOccurred())
	//			Expect(configMap.Data["lavinmq.ini"]).To(Equal(oldConfigMap.Data["lavinmq.ini"]))
	//		})
	//	})
})
