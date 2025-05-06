package resource_utils

import (
	corev1 "k8s.io/api/core/v1"
)

func EqualResourceRequirements(a, b corev1.ResourceRequirements) bool {
	if !equalResourceLists(a.Requests, b.Requests) {
		return false
	}
	return equalResourceLists(a.Limits, b.Limits)
}

func equalResourceLists(a, b corev1.ResourceList) bool {
	if a.Cpu().Cmp(*b.Cpu()) != 0 {
		return false
	}
	if a.Memory().Cmp(*b.Memory()) != 0 {
		return false
	}
	return true
}
