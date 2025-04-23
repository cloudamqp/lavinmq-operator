package reconciler_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	lavinmqv1alpha1 "lavinmq-operator/api/v1alpha1"
	"lavinmq-operator/internal/reconciler"
)

func TestStatefulSetReconciler(t *testing.T) {
	instance := &lavinmqv1alpha1.LavinMQ{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lavinmq",
			Namespace: "default",
		},
		Spec: lavinmqv1alpha1.LavinMQSpec{
			Replicas: 1,
			Image:    "test-image:latest",
			DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{ // Default spec
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		},
	}

	rc := &reconciler.StatefulSetReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	err := k8sClient.Create(t.Context(), instance)

	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	instance.Spec.Image = "test-image:latest2"
	err = k8sClient.Update(t.Context(), instance)
	if err != nil {
		t.Fatalf("Failed to update instance: %v", err)
	}

	_, err = rc.Reconcile(t.Context())
	if err != nil {
		t.Fatalf("Failed to reconcile instance: %v", err)
	}

	sts := &appsv1.StatefulSet{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sts)
	if err != nil {
		t.Fatalf("Failed to get statefulset: %v", err)
	}

	if sts.Spec.Template.Spec.Containers[0].Image != "test-image:latest2" {
		t.Fatalf("Statefulset image not updated")
	}
}
