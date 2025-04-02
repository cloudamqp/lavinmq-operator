package builder

import (
	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ini "gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	var (
		resourceName = "test-resource"
		instance     = &cloudamqpcomv1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: "default",
			},
		}
		builder ServiceConfigBuilder
	)

	BeforeEach(func() {
		builder = ServiceConfigBuilder{
			ResourceBuilder: &ResourceBuilder{
				Instance: instance,
				Scheme:   scheme.Scheme,
			},
		}
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
			obj, err := builder.Build()
			configMap := obj.(*corev1.ConfigMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Name).To(Equal("test-resource-config"))
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
			obj, err := builder.Build()
			configMap := obj.(*corev1.ConfigMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Name).To(Equal("test-resource-config"))
			verifyConfigMapEquality(configMap, expectedConfig)
		})

	})

	Context("When diffing config maps", func() {
		var (
			oldConfigMap *corev1.ConfigMap
			newConfigMap *corev1.ConfigMap
		)

		BeforeEach(func() {
			oldConfigMap = &corev1.ConfigMap{
				Data: map[string]string{
					"lavinmq.ini": `
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
advertised_uri = tcp://test-resource:5679`,
				},
			}
			newConfigMap = &corev1.ConfigMap{
				Data: map[string]string{
					"lavinmq.ini": `
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
advertised_uri = tcp://test-resource:5679`,
				},
			}
		})

		It("should return no diff when configs are identical", func() {
			result, diff, err := builder.Diff(oldConfigMap, newConfigMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(diff).To(BeFalse())
			Expect(result).To(Equal(oldConfigMap))
		})

		It("should return a diff when config content changes", func() {
			newConfigMap.Data["lavinmq.ini"] = `
[main]
log_level = debug
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
advertised_uri = tcp://test-resource:5679`

			result, diff, err := builder.Diff(oldConfigMap, newConfigMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(diff).To(BeTrue())
			Expect(result.(*corev1.ConfigMap).Data["lavinmq.ini"]).To(Equal(newConfigMap.Data["lavinmq.ini"]))
		})
	})
})
