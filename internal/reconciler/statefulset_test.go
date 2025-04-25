package reconciler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"lavinmq-operator/internal/reconciler"
	testutils "lavinmq-operator/internal/test_utils"
)

func TestStatefulSetReconciler(t *testing.T) {
	t.Parallel()
	instance := testutils.GetDefaultInstance(&testutils.DefaultInstanceSettings{})

	err := testutils.CreateNamespace(t.Context(), k8sClient, instance.Namespace)
	assert.NoErrorf(t, err, "Failed to create namespace")
	defer testutils.DeleteNamespace(t.Context(), k8sClient, instance.Namespace)

	rc := &reconciler.StatefulSetReconciler{
		ResourceReconciler: &reconciler.ResourceReconciler{
			Instance: instance,
			Scheme:   scheme.Scheme,
			Client:   k8sClient,
		},
	}

	err = k8sClient.Create(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to create instance")

	instance.Spec.Image = "test-image:latest2"
	err = k8sClient.Update(t.Context(), instance)
	assert.NoErrorf(t, err, "Failed to update instance")

	_, err = rc.Reconcile(t.Context())
	assert.NoErrorf(t, err, "Failed to reconcile instance")

	sts := &appsv1.StatefulSet{}
	err = k8sClient.Get(t.Context(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, sts)
	assert.NoErrorf(t, err, "Failed to get statefulset")

	assert.Equal(t, "test-image:latest2", sts.Spec.Template.Spec.Containers[0].Image)
}
