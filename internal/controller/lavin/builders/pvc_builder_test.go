package builder

import (
	"lavinmq-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("PVCBuilder", func() {
	var (
		instance *v1alpha1.LavinMQ
		builder  *PVCBuilder
	)

	BeforeEach(func() {
		instance = &v1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-lavinmq",
				Namespace: "default",
			},
		}
		builder = &PVCBuilder{
			ResourceBuilder: &ResourceBuilder{
				Instance: instance,
				Scheme:   scheme.Scheme,
			},
		}
	})

	Describe("Build", func() {
		It("should create a PVC with the correct template", func() {
			instance.Spec.DataVolumeClaimSpec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10Gi"),
					},
				},
			}

			pvc, err := builder.Build()

			Expect(err).NotTo(HaveOccurred())
			Expect(pvc).NotTo(BeNil())

			pvcObj := pvc.(*corev1.PersistentVolumeClaim)
			Expect(pvcObj.Name).To(Equal(instance.Name))
			Expect(pvcObj.Namespace).To(Equal(instance.Namespace))
			Expect(pvcObj.Spec).To(Equal(instance.Spec.DataVolumeClaimSpec))
		})
	})

	Describe("Diff", func() {
		Context("when there are no changes", func() {
			It("should return no diff and no error", func() {
				oldPVC := &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
					},
				}
				newPVC := &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
					},
				}

				result, diff, err := builder.Diff(oldPVC, newPVC)

				Expect(err).NotTo(HaveOccurred())
				Expect(diff).To(BeFalse())
				Expect(result).NotTo(BeNil())
			})
		})

		Context("when storage class changes", func() {
			It("should return an error", func() {
				oldPVC := &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						StorageClassName: stringPtr("old-storage-class"),
					},
				}
				newPVC := &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						StorageClassName: stringPtr("new-storage-class"),
					},
				}

				_, _, err := builder.Diff(oldPVC, newPVC)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("storage class change not supported"))
			})
		})

		Context("when access modes change", func() {
			It("should return an error", func() {
				oldPVC := &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					},
				}
				newPVC := &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					},
				}

				_, diff, err := builder.Diff(oldPVC, newPVC)

				Expect(err).NotTo(HaveOccurred())
				Expect(diff).To(BeFalse())
			})
		})

		Context("when storage size changes", func() {
			Context("when storage size increases", func() {
				It("should allow the change and return a diff", func() {
					oldPVC := &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
						},
					}
					newPVC := &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("20Gi"),
								},
							},
						},
					}

					result, diff, err := builder.Diff(oldPVC, newPVC)

					Expect(err).NotTo(HaveOccurred())
					Expect(diff).To(BeTrue())
					Expect(result).NotTo(BeNil())
					Expect(result.(*corev1.PersistentVolumeClaim).Spec.Resources.Requests).
						To(Equal(newPVC.Spec.Resources.Requests))
				})
			})

			Context("when storage size decreases", func() {
				It("should return an error", func() {
					oldPVC := &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("20Gi"),
								},
							},
						},
					}
					newPVC := &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
						},
					}

					_, _, err := builder.Diff(oldPVC, newPVC)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("volume size decreased, not supported"))
				})
			})
		})
	})
})

func stringPtr(s string) *string {
	return &s
}
