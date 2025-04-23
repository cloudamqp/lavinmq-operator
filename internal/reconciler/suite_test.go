package reconciler_test

import (
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"lavinmq-operator/api/v1alpha1"
	testutils "lavinmq-operator/internal/test_utils"
	// +kubebuilder:scaffold:imports
)

var testEnv *envtest.Environment
var k8sClient client.Client

func TestMain(m *testing.M) {
	testEnv, k8sClient = testutils.StartKubeTestEnv()

	code := m.Run()

	logf.Log.Info("Tearing down test suite")

	err := testEnv.Stop()
	if err != nil {
		logf.Log.Error(err, "Failed to stop test environment")
		os.Exit(1)
	}

	os.Exit(code)
}

var defaultInstance = &v1alpha1.LavinMQ{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-default",
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
