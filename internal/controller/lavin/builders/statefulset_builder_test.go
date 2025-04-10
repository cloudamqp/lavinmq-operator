package builder

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lavinmqv1alpha1 "lavinmq-operator/api/v1alpha1"
)

var _ = Describe("StatefulSetBuilder", func() {
	var (
		instance *lavinmqv1alpha1.LavinMQ
		b        *ResourceBuilder
		sut      *StatefulSetBuilder // System Under Test
		scheme   *runtime.Scheme
		log      logr.Logger

		oldSts *appsv1.StatefulSet
		newSts *appsv1.StatefulSet
	)

	BeforeEach(func() {
		log = logr.Discard() // Use discard logger for tests unless debugging
		scheme = runtime.NewScheme()
		Expect(lavinmqv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(appsv1.AddToScheme(scheme)).To(Succeed())

		instance = &lavinmqv1alpha1.LavinMQ{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-lavinmq",
				Namespace: "test-ns",
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

		b = &ResourceBuilder{
			Instance: instance,
			Scheme:   scheme,
			Logger:   log,
		}
		sut = b.StatefulSetBuilder(client.Client(nil)) // Get the specific builder we are testing

		// Create base statefulsets for modification in Contexts
		baseSts, err := sut.Build() // Build generates the 'new'/'desired' state based on instance
		Expect(err).NotTo(HaveOccurred())
		newSts = baseSts.(*appsv1.StatefulSet) // This is our initial 'desired' state

		// Create a copy for the 'old' state, initially identical
		oldSts = newSts.DeepCopy()
	})

	Describe("Diff", func() {
		var (
			diffResult client.Object
			changed    bool
			err        error
		)

		// Helper to create a PVC Spec
		createPVCSpec := func(storageSize string, storageClassName *string, accessModes []corev1.PersistentVolumeAccessMode) corev1.PersistentVolumeClaimSpec {
			return corev1.PersistentVolumeClaimSpec{
				AccessModes: accessModes,
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(storageSize),
					},
				},
				StorageClassName: storageClassName,
			}
		}

		// Helper to set PVC Spec on a StatefulSet
		setPVCSpec := func(sts *appsv1.StatefulSet, spec corev1.PersistentVolumeClaimSpec) {
			// Ensure VolumeClaimTemplates exists and has at least one item
			if len(sts.Spec.VolumeClaimTemplates) == 0 {
				sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
					{ObjectMeta: metav1.ObjectMeta{Name: "data"}}, // Make sure name matches builder
				}
			}
			sts.Spec.VolumeClaimTemplates[0].Spec = spec
		}

		Context("when VolumeClaimTemplates differ", func() {
			var (
				oldSpec corev1.PersistentVolumeClaimSpec
				newSpec corev1.PersistentVolumeClaimSpec
			)

			JustBeforeEach(func() {
				// Set the specs on the old and new statefulsets
				setPVCSpec(oldSts, oldSpec)
				setPVCSpec(newSts, newSpec) // newSts reflects the desired state

				// Perform the diff operation
				diffResult, changed, err = sut.Diff(oldSts, newSts)
			})

			Context("due to storage size change", func() {
				BeforeEach(func() {
					oldSpec = createPVCSpec("1Gi", nil, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce})
					newSpec = createPVCSpec("2Gi", nil, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}) // Increased size
				})

				It("should return changed = true", func() {
					Expect(changed).To(BeTrue())
				})

				It("should return the original object with VCT unchanged (BUG)", func() {
					Expect(err).NotTo(HaveOccurred())
					returnedSts := diffResult.(*appsv1.StatefulSet)
					Expect(returnedSts.Spec.VolumeClaimTemplates[0].Spec).To(Equal(oldSpec), "BUG: The diff logic detects the change but doesn't update the old object's VolumeClaimTemplate spec before returning.")
					Expect(returnedSts.Spec.VolumeClaimTemplates[0].Spec).NotTo(Equal(newSpec))
				})
			})

			Context("due to storage class change", func() {
				BeforeEach(func() {
					oldClassName := "standard"
					newClassName := "premium"
					oldSpec = createPVCSpec("1Gi", &oldClassName, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce})
					newSpec = createPVCSpec("1Gi", &newClassName, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}) // Changed class
				})

				It("should return changed = true", func() {
					Expect(changed).To(BeTrue())
				})

				It("should return the original object with VCT unchanged (BUG)", func() {
					Expect(err).NotTo(HaveOccurred())
					returnedSts := diffResult.(*appsv1.StatefulSet)
					Expect(returnedSts.Spec.VolumeClaimTemplates[0].Spec).To(Equal(oldSpec), "BUG: The diff logic detects the change but doesn't update the old object's VolumeClaimTemplate spec before returning.")
					Expect(returnedSts.Spec.VolumeClaimTemplates[0].Spec).NotTo(Equal(newSpec))
				})
			})

			Context("due to access mode change", func() {
				BeforeEach(func() {
					oldSpec = createPVCSpec("1Gi", nil, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce})
					newSpec = createPVCSpec("1Gi", nil, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}) // Changed access mode
				})

				It("should return changed = true", func() {
					Expect(changed).To(BeTrue())
				})

				It("should return the original object with VCT unchanged (BUG)", func() {
					Expect(err).NotTo(HaveOccurred())
					returnedSts := diffResult.(*appsv1.StatefulSet)
					Expect(returnedSts.Spec.VolumeClaimTemplates[0].Spec).To(Equal(oldSpec), "BUG: The diff logic detects the change but doesn't update the old object's VolumeClaimTemplate spec before returning.")
					Expect(returnedSts.Spec.VolumeClaimTemplates[0].Spec).NotTo(Equal(newSpec))
				})
			})
		})

		Context("when VolumeClaimTemplates are identical", func() {
			BeforeEach(func() {
				// Ensure both old and new have the same, non-default spec for thoroughness
				identicalSpec := createPVCSpec("5Gi", Ptr("fast-ssd"), []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce})
				setPVCSpec(oldSts, identicalSpec)
				setPVCSpec(newSts, identicalSpec)

				// Perform the diff operation
				diffResult, changed, err = sut.Diff(oldSts, newSts)
			})

			It("should return changed = false", func() {
				Expect(changed).To(BeFalse())
			})

			It("should return the original object unmodified", func() {
				Expect(err).NotTo(HaveOccurred())
				returnedSts := diffResult.(*appsv1.StatefulSet)
				Expect(returnedSts).To(Equal(oldSts)) // Pointer comparison might be too strict, let's check spec
				Expect(returnedSts.Spec.VolumeClaimTemplates[0].Spec).To(Equal(oldSts.Spec.VolumeClaimTemplates[0].Spec))
			})
		})

		// Add more contexts for other parts of Diff (Replicas, Template Spec) if needed
	})
})

// Helper function to get a pointer to a string
func Ptr(s string) *string {
	return &s
}
