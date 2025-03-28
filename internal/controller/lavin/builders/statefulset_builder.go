package builder

import (
	"fmt"
	"reflect"

	"lavinmq-operator/internal/controller/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatefulSetBuilder struct {
	*ResourceBuilder
}

func (builder *ResourceBuilder) StatefulSetBuilder() *StatefulSetBuilder {
	return &StatefulSetBuilder{
		ResourceBuilder: builder,
	}
}

func (b *StatefulSetBuilder) Name() string {
	return "statefulset"
}

func (b *StatefulSetBuilder) NewObject() client.Object {
	labels := utils.LabelsForLavinMQ(b.Instance)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.Instance.Name,
			Namespace: b.Instance.Namespace,
			Labels:    labels,
		},
	}
}

func (b *StatefulSetBuilder) Build() (client.Object, error) {
	statefulset := b.baseStatefulSet()

	b.appendTlsConfig(statefulset)

	return statefulset, nil
}

func (b *StatefulSetBuilder) baseStatefulSet() *appsv1.StatefulSet {
	statefulset := b.NewObject().(*appsv1.StatefulSet)
	configVolumeName := fmt.Sprintf("%s-config", b.Instance.Name)

	statefulset.Spec = appsv1.StatefulSetSpec{
		Replicas: &b.Instance.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: statefulset.Labels,
		},
		ServiceName: fmt.Sprintf("%s-service", b.Instance.Name),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: statefulset.Labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "lavinmq",
						Image: b.Instance.Spec.Image,
						Ports: b.Instance.Spec.Ports,
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/lavinmq",
							},
							{
								Name:      configVolumeName,
								MountPath: "/etc/lavinmq",
								ReadOnly:  true,
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: b.Instance.Name,
							},
						},
					},
					{
						Name: configVolumeName,
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: configVolumeName},
							},
						},
					},
				},
			},
		},
	}

	return statefulset
}

func (b *StatefulSetBuilder) appendTlsConfig(statefulset *appsv1.StatefulSet) {
	if b.Instance.Spec.Secrets == nil {
		return
	}

	statefulset.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		statefulset.Spec.Template.Spec.Containers[0].VolumeMounts,
		corev1.VolumeMount{
			Name:      "tls",
			MountPath: "/etc/lavinmq/tls",
			ReadOnly:  true,
		},
	)
	statefulset.Spec.Template.Spec.Volumes = append(
		statefulset.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: "tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: b.Instance.Spec.Secrets[0].Name,
				},
			},
		},
	)
}

func (b *StatefulSetBuilder) Diff(old, new client.Object) (client.Object, bool, error) {
	logger := b.Logger
	oldSts := old.(*appsv1.StatefulSet)
	newSts := new.(*appsv1.StatefulSet)
	changed := false

	//	'replicas', 'ordinals', 'template', 'updateStrategy',
	// 'persistentVolumeClaimRetentionPolicy' and 'minReadySeconds',

	if *oldSts.Spec.Replicas != *newSts.Spec.Replicas {
		logger.Info("Replicas changed", "old", oldSts.Spec.Replicas, "new", newSts.Spec.Replicas)
		// TODO: Add support for scaling.
	}

	if !reflect.DeepEqual(oldSts.Spec.Template, newSts.Spec.Template) {
		logger.Info("Template changed, updating")
		oldSts.Spec.Template = newSts.Spec.Template
		changed = true
	}

	// TODO: Do we need to do a disk check here now that we have a PVC?

	return oldSts, changed, nil
}
