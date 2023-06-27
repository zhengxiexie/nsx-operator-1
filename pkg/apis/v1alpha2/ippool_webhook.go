/* Copyright © 2023 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package v1alpha2

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var ippoollog = logf.Log.WithName("ippool-resource")

func (r *IPPool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nsx-vmware-com-v1alpha2-ippool,mutating=false,failurePolicy=fail,sideEffects=None,groups=nsx.vmware.com,resources=ippools,verbs=create;update,versions=v1alpha2,name=vippool.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &IPPool{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *IPPool) ValidateCreate() error {
	ippoollog.Info("validate create", "name", r.Name)
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *IPPool) ValidateUpdate(old runtime.Object) error {
	ippoollog.Info("validate update", "name", r.Name)

	// IPPool.IPPoolSpec.SubnetRequest.PrefixLength is immutable
	oldIPPool := old.(*IPPool)
	oldIPPoolSubnetsMap := make(map[string]int)
	newIPPoolSubnetsMap := make(map[string]int)
	for _, subnet := range oldIPPool.Spec.Subnets {
		oldIPPoolSubnetsMap[subnet.Name] = subnet.PrefixLength
	}
	for _, subnet := range r.Spec.Subnets {
		newIPPoolSubnetsMap[subnet.Name] = subnet.PrefixLength
	}
	for name, oldPrefixLength := range oldIPPoolSubnetsMap {
		if newPrefixLength, ok := newIPPoolSubnetsMap[name]; ok {
			if oldPrefixLength != newPrefixLength {
				return field.Invalid(field.NewPath("spec").Child("subnets"), r.Spec.Subnets,
					"IPPool.IPPoolSpec.SubnetRequest.PrefixLength is immutable")
			}
		}
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *IPPool) ValidateDelete() error {
	ippoollog.Info("validate delete", "name", r.Name)
	return nil
}
