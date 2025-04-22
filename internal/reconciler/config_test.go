package reconciler_test

import (
	"context"
	"fmt"
	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"
	"lavinmq-operator/internal/reconciler"

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

var _ = Describe("ConfigReconciler", func() {
	var namespacedName = types.NamespacedName{
		Name:      "test-resource",
		Namespace: "default",
	}
	var (
		instance *cloudamqpcomv1alpha1.LavinMQ
		rc       *reconciler.ConfigReconciler
	)

	BeforeEach(func() {
		instance = &cloudamqpcomv1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}

		rc = &reconciler.ConfigReconciler{
			ResourceReconciler: &reconciler.ResourceReconciler{
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
			data_dir = /var/lib/lavinmq

			[mgmt]
			bind = 0.0.0.0
			port = 15672

			[amqp]
			bind = 0.0.0.0
			port = 5672

			[mqtt]
			bind = 0.0.0.0
			port = 1883

			[clustering]
			bind = 0.0.0.0
			port = 5679
	`
		It("Should create a default ConfigMap", func() {
			rc.Reconcile(context.Background())

			configMap := &corev1.ConfigMap{}
			err := k8sClient.Get(context.Background(), namespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Name).To(Equal(namespacedName.Name))
			verifyConfigMapEquality(configMap, expectedConfig)
		})
	})

	Context("When providing custom ports", func() {
		BeforeEach(func() {
			instance.Spec.Config.Amqp.Port = 1111
			instance.Spec.Config.Mgmt.Port = 2222
			instance.Spec.Config.Amqp.TlsPort = 3333
			instance.Spec.Config.Mgmt.TlsPort = 4444
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
		})

		expectedConfig := `
			[main]
			data_dir = /var/lib/lavinmq

			[mgmt]
			bind = 0.0.0.0
			port = 2222
			tls_port = 4444

			[amqp]
			bind = 0.0.0.0
			port = 1111
			tls_port = 3333

			[mqtt]
			bind = 0.0.0.0
			port = 1883

			[clustering]
			bind = 0.0.0.0
			port = 5679
		`

		It("Should setup ports in according section", func() {
			rc.Reconcile(context.Background())

			configMap := &corev1.ConfigMap{}
			err := k8sClient.Get(context.Background(), namespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Name).To(Equal(namespacedName.Name))
			verifyConfigMapEquality(configMap, expectedConfig)
		})

	})

	Context("When disabling non tls ports", func() {
		BeforeEach(func() {
			instance.Spec.Config.Amqp.Port = -1
			instance.Spec.Config.Mgmt.Port = -1
			instance.Spec.Config.Mqtt.Port = -1
			Expect(k8sClient.Update(context.Background(), instance)).To(Succeed())
		})

		expectedConfig := `
			[main]
			data_dir = /var/lib/lavinmq

			[mgmt]
			bind = 0.0.0.0
			port = -1

			[amqp]
			bind = 0.0.0.0
			port = -1

			[mqtt]
			bind = 0.0.0.0
			port = -1

			[clustering]
			bind = 0.0.0.0
			port = 5679
		`

		It("Should remove ports in according section", func() {
			rc.Reconcile(context.Background())

			configMap := &corev1.ConfigMap{}
			err := k8sClient.Get(context.Background(), namespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Name).To(Equal(namespacedName.Name))
			fmt.Printf("ConfigMap Data: %v\n", configMap.Data)
			fmt.Printf("Expected Config: %v\n", expectedConfig)
			verifyConfigMapEquality(configMap, expectedConfig)
		})
	})
})
