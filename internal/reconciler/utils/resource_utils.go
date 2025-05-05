package resource_utils

import (
	corev1 "k8s.io/api/core/v1"
)

// equalResourceRequirements compares two ResourceRequirements objects using the Cmp method for each resource
func EqualResourceRequirements(a, b corev1.ResourceRequirements) bool {
	// Compare Requests
	if !equalResourceLists(a.Requests, b.Requests) {
		return false
	}

	// Compare Limits
	if !equalResourceLists(a.Limits, b.Limits) {
		return false
	}

	return true
}

func equalResourceLists(a, b corev1.ResourceList) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, exists := b[k]; !exists || v.Cmp(bv) != 0 {
			return false
		}
	}

	return true
}
