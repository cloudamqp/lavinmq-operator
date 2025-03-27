package utils

import (
	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"
)

func LabelsForLavinMQ(instance *cloudamqpcomv1alpha1.LavinMQ) map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/name":       "lavinmq-operator",
		"app.kubernetes.io/managed-by": "LavinMQController",
	}

	// Append instance labels
	for k, v := range instance.Labels {
		labels[k] = v
	}

	return labels
}
