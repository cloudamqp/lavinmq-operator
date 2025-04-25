package reconciler_test

import (
	"os"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

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
