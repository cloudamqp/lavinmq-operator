package builder

import (
	"context"

	cloudamqpcomv1alpha1 "lavinmq-operator/api/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceBuilder struct {
	Instance *cloudamqpcomv1alpha1.LavinMQ
	Scheme   *runtime.Scheme
	Context  context.Context
}

type Builder interface {
	NewObject() client.Object
	Build() (client.Object, error)
	Diff(old, new client.Object) (client.Object, bool, error)
	Name() string
}

func (builder *ResourceBuilder) Builders() []Builder {
	return []Builder{
		builder.ConfigBuilder(),
		builder.StatefulSetBuilder(),
	}
}
