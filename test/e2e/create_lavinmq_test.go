package e2e

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	cloudamqpcomv1alpha1 "github.com/cloudamqp/lavinmq-operator/api/v1alpha1"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestCreateLavinMQ(t *testing.T) {
	instanceName := "lavinmq"
	stsObj := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
	}
	feature := features.New("Create LavinMQ").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			assert.NoError(t, err)

			err = cloudamqpcomv1alpha1.AddToScheme(r.GetScheme())
			assert.NoError(t, err)

			t.Logf("Trying to create LavinMQ in namespace %s", namespace)
			r.WithNamespace(namespace)

			lavinmq := &cloudamqpcomv1alpha1.LavinMQ{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "cloudamqp.com/v1alpha1",
					Kind:       "LavinMQ",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
				},
				Spec: cloudamqpcomv1alpha1.LavinMQSpec{
					Replicas: 1,
					Image:    "cloudamqp/lavinmq:2.2.0",
					EtcdEndpoints: []string{
						fmt.Sprintf("etcd-cluster-0.etcd-cluster.%s.svc.cluster.local:2379", namespace),
						fmt.Sprintf("etcd-cluster-1.etcd-cluster.%s.svc.cluster.local:2379", namespace),
						fmt.Sprintf("etcd-cluster-2.etcd-cluster.%s.svc.cluster.local:2379", namespace),
					},
					Config: cloudamqpcomv1alpha1.LavinMQConfig{
						Main: cloudamqpcomv1alpha1.MainConfig{
							ConsumerTimeout: 20000,
						},
					},
					DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("3Gi"),
							},
						},
					},
				},
			}

			err = r.Create(ctx, lavinmq)
			assert.NoErrorf(t, err, "Failed to create LavinMQ")

			err = r.Get(ctx, lavinmq.ObjectMeta.GetName(), namespace, lavinmq)
			assert.NoErrorf(t, err, "Failed to get LavinMQ")

			return ctx
		}).
		Assess("Check if LavinMQ starts", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			assert.NoError(t, err)

			r.WithNamespace(namespace)

			err = cloudamqpcomv1alpha1.AddToScheme(r.GetScheme())
			assert.NoError(t, err)

			lavinmqSts := &appsv1.StatefulSet{}

			err = wait.For(conditions.New(r).ResourceMatch(stsObj, func(object k8s.Object) bool {
				err := r.Get(ctx, instanceName, namespace, lavinmqSts)
				return err == nil
			}), wait.WithTimeout(time.Minute), wait.WithInterval(time.Second*5))

			assert.NoErrorf(t, err, "Failed to get LavinMQ StatefulSet")

			err = wait.For(conditions.New(r).ResourceScaled(lavinmqSts, func(object k8s.Object) int32 {
				return lavinmqSts.Status.ReadyReplicas
			}, 1), wait.WithTimeout(time.Minute), wait.WithInterval(time.Second*5))

			assert.NoErrorf(t, err, "Failed to scale LavinMQ StatefulSet")

			lavinPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-0", instanceName),
					Namespace: namespace,
				},
			}

			err = r.Get(ctx, lavinPod.ObjectMeta.GetName(), namespace, lavinPod)
			assert.NoErrorf(t, err, "Failed to get LavinMQ pod")

			err = wait.For(conditions.New(r).PodRunning(lavinPod), wait.WithTimeout(time.Minute), wait.WithInterval(time.Second*5))
			assert.NoErrorf(t, err, "Failed to wait for LavinMQ pod to be running")

			var stdout, stderr bytes.Buffer
			err = r.ExecInPod(ctx, namespace, lavinPod.Name, lavinPod.Spec.Containers[0].Name, []string{"lavinmqctl", "status"}, &stdout, &stderr)
			assert.NoErrorf(t, err, "Failed to execute lavinmqctl status", stderr.String())

			if !strings.Contains(stdout.String(), fmt.Sprintf("%s-0", instanceName)) {
				assert.FailNowf(t, "LavinMQ is not running", stdout.String())
			}

			return ctx
		}).Feature()

	testEnv.Test(t, feature)
}
