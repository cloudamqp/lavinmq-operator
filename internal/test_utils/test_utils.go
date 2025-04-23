package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func StartKubeTestEnv() (*envtest.Environment, client.Client) {
	logf.Log.Info("Setting up test suite")

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err := testEnv.Start()
	if err != nil {
		logf.Log.Error(err, "Failed to start test environment")
		os.Exit(1)
	}

	err = cloudamqpcomv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		logf.Log.Error(err, "Failed to add scheme")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:scheme
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		logf.Log.Error(err, "Failed to create k8s client")
		os.Exit(1)
	}

	return testEnv, k8sClient
}
