/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var lavinmqlog = logf.Log.WithName("lavinmq-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *LavinMQ) SetupWebhookWithManager(mgr ctrl.Manager) error {

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-cloudamqp-com-v1alpha1-lavinmq,mutating=false,failurePolicy=fail,sideEffects=None,groups=cloudamqp.com,resources=lavinmqs,verbs=create;update,versions=v1alpha1,name=vlavinmq.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &LavinMQ{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *LavinMQ) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	lavinmqlog.Info("validate create", "name", r.Name)
	lavinmqlog.Info("validate create", r)
	return nil, fmt.Errorf("LavinMQ %s is not ready to be created", r.Name)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LavinMQ) ValidateUpdate(ctx context.Context, obj runtime.Object, old runtime.Object) (admission.Warnings, error) {
	lavinmqlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *LavinMQ) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	lavinmqlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
