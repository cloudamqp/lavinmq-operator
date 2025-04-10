package builder

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"lavinmq-operator/internal/controller/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatefulSetReconciler struct {
	*ResourceBuilder
}

func (builder *ResourceBuilder) StatefulSetReconciler() *StatefulSetReconciler {
	return &StatefulSetReconciler{
		ResourceBuilder: builder,
	}
}

func (b *StatefulSetReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	statefulset := b.newObject()

	err := b.GetItem(ctx, statefulset)
	if err != nil {
		if apierrors.IsNotFound(err) {
			b.CreateItem(ctx, statefulset)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	err = b.updateFields(ctx, statefulset)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = b.Client.Update(ctx, statefulset)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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
				Labels:      statefulset.Labels,
				Annotations: make(map[string]string),
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
								Name:      "data",
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
					Name: "data",
				},
				Spec: b.Instance.Spec.DataVolumeClaimSpec,
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
	if b.Instance.Spec.TlsSecret == nil {
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
					SecretName: b.Instance.Spec.TlsSecret.Name,
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
		oldSts.Spec.Replicas = newSts.Spec.Replicas
		changed = true
	}

	if diff, err := b.diffTemplate(&oldSts.Spec.Template.Spec, &newSts.Spec.Template.Spec); err != nil {
		return nil, false, err
	} else if diff {
		changed = true
	}

	// TODO: Do we need to do a disk check here now that we have a PVC?

	return oldSts, changed, nil
}

func (b *StatefulSetBuilder) diffPersistentVolumeClaim() (bool, error) {
	changed := false
	for i := 0; i < int(b.Instance.Spec.Replicas); i++ {
		pvc := &corev1.PersistentVolumeClaim{}
		err := b.Client.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("data-%s-%d", b.Instance.Name, i), Namespace: b.Instance.Namespace}, pvc)
		if err != nil {
			return false, err
		}

		comp := pvc.Spec.Resources.Requests.Storage().Cmp(*b.Instance.Spec.DataVolumeClaimSpec.Resources.Requests.Storage())

		if comp == -1 {
			changed = true
			pvc.Spec.Resources.Requests = b.Instance.Spec.DataVolumeClaimSpec.Resources.Requests
			err = b.Client.Update(context.Background(), pvc)
			if err != nil {
				return false, err
			}

			b.Logger.Info("PVC size increased", "pvc", pvc.Name)
		}
	}

	return changed, nil
}

func (b *StatefulSetBuilder) diffTemplate(old, new *corev1.PodSpec) (bool, error) {
	changed := false
	if len(old.Containers) != len(new.Containers) && len(old.Containers) != 1 {
		return false, fmt.Errorf("container count mismatch, expects 1")
	}

	// Pointer the old as that's the object we're mutating
	oldContainer := &old.Containers[0]
	newContainer := new.Containers[0]

	if oldContainer.Image != newContainer.Image {
		oldContainer.Image = newContainer.Image
		changed = true
	}

	// TODO: Expand this to own methods and granular checks
	if !reflect.DeepEqual(oldContainer.Args, newContainer.Args) {
		oldContainer.Args = newContainer.Args
		changed = true
	}

	// TODO: Expand this to own methods and granular checks
	if !reflect.DeepEqual(oldContainer.Ports, newContainer.Ports) {
		oldContainer.Ports = newContainer.Ports
		changed = true
	}

	index := slices.IndexFunc(old.Volumes, func(v corev1.Volume) bool {
		return v.Name == "tls"
	})

	if index != -1 {
		secretName := old.Volumes[index].VolumeSource.Secret.SecretName
		// Checks if the secret name is the same as the one in the instance spec
		if b.Instance.Spec.TlsSecret != nil && b.Instance.Spec.TlsSecret.Name != secretName {
			changed = true
		}
	}

	if len(old.Volumes) != len(new.Volumes) {
		changed = true
	}

	old.Volumes = new.Volumes

	if changed {
		b.Logger.Info("Template changed, updating")
	}

	return changed, nil
}
