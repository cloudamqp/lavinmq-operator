package builder

import (
	"fmt"
	"reflect"

	"lavinmq-operator/internal/controller/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

	volume := b.Instance.Spec.DataVolumeClaimSpec
	volumeName := fmt.Sprintf("%s-data", b.Instance.Name)
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
						Name:    "lavinmq",
						Image:   b.Instance.Spec.Image,
						Command: []string{"/usr/bin/lavinmq"},
						Args:    b.cliArgs(),
						Ports:   b.Instance.Spec.Ports,
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      volumeName,
								MountPath: "/var/lib/lavinmq",
							},
							{
								Name:      configVolumeName,
								MountPath: "/etc/lavinmq",
								ReadOnly:  true,
							},
						},
						Env: []corev1.EnvVar{
							{
								Name: "POD_NAME",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
							{
								Name: "POD_NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
						},
					},
				},
				Volumes: []corev1.Volume{
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
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: volumeName,
				},
				Spec: volume,
			},
		},
	}

	return statefulset
}

func (b *StatefulSetBuilder) cliArgs() []string {
	defaultArgs := []string{
		"--bind=0.0.0.0",
		"--guest-only-loopback=false",
	}

	if b.Instance.Spec.Replicas > 0 {
		// Clustering config is currently spread between CLI here and in the config file.
		clusteringArgs := []string{
			fmt.Sprintf("--clustering-advertised-uri=tcp://$(POD_NAME).%s-service.$(POD_NAMESPACE).svc.cluster.local:5679", b.Instance.Name),
		}
		defaultArgs = append(defaultArgs, clusteringArgs...)
	}

	return defaultArgs
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
	logger := log.FromContext(b.Context)
	oldSts := old.(*appsv1.StatefulSet)
	newSts := new.(*appsv1.StatefulSet)
	changed := false

	//	'replicas', 'ordinals', 'template', 'updateStrategy',
	// 'persistentVolumeClaimRetentionPolicy' and 'minReadySeconds',

	if *oldSts.Spec.Replicas != *newSts.Spec.Replicas {
		logger.Info("Replicas changed", "old", oldSts.Spec.Replicas, "new", newSts.Spec.Replicas)
		// TODO: Add support for scaling.
		oldSts.Spec.Replicas = newSts.Spec.Replicas
		changed = true
	}

	if !reflect.DeepEqual(oldSts.Spec.Template, newSts.Spec.Template) {
		logger.Info("Template changed, updating")
		oldSts.Spec.Template = newSts.Spec.Template
		changed = true
	}

	if oldSts.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests.Storage().Cmp(*newSts.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests.Storage()) != 0 {
		logger.Info("VolumeClaimTemplates changed, updating")
		oldSts.Spec.VolumeClaimTemplates = newSts.Spec.VolumeClaimTemplates
		changed = true
	}

	return oldSts, changed, nil
}
