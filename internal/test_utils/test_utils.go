package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"

	cloudamqpcomv1alpha1 "github.com/cloudamqp/lavinmq-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
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

type DefaultInstanceSettings struct {
	Name      *string
	Namespace *string
	Replicas  *int32
	Storage   *string
	Image     *string
	Resources corev1.ResourceRequirements
}

func GetDefaultInstance(settings *DefaultInstanceSettings) *cloudamqpcomv1alpha1.LavinMQ {
	// --- Defaults ---
	defaultName := envconf.RandomName("name", 10)
	defaultNamespace := envconf.RandomName("namespace", 15)
	defaultReplicas := int32(1) // Need to explicitly cast int to int32
	defaultStorage := "10Gi"
	defaultImage := "cloudamqp/lavinmq:2.4.1"
	defaultResources := corev1.ResourceRequirements{}
	// --- --- --- --- --

	instanceName := defaultName
	if settings.Name != nil {
		instanceName = *settings.Name
	}

	instanceNamespace := defaultNamespace
	if settings.Namespace != nil {
		instanceNamespace = *settings.Namespace
	}

	instanceReplicas := defaultReplicas
	if settings.Replicas != nil {
		instanceReplicas = *settings.Replicas
	}

	instanceStorage := defaultStorage
	if settings.Storage != nil {
		instanceStorage = *settings.Storage
	}

	instanceImage := defaultImage
	if settings.Image != nil {
		instanceImage = *settings.Image
	}

	if !reflect.DeepEqual(settings.Resources, corev1.ResourceRequirements{}) {
		defaultResources = settings.Resources
	}

	return &cloudamqpcomv1alpha1.LavinMQ{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: instanceNamespace,
		},
		Spec: cloudamqpcomv1alpha1.LavinMQSpec{
			Image:     instanceImage,
			Resources: defaultResources,
			DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(instanceStorage),
					},
				},
			},
			Replicas: instanceReplicas,
		},
	}
}

func CreateNamespace(ctx context.Context, client client.Client, namespace string) error {
	return client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
}

func DeleteNamespace(ctx context.Context, client client.Client, namespace string) error {
	return client.Delete(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
}
