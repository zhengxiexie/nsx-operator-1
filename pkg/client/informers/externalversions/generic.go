/* Copyright © 2024 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	v1alpha1 "github.com/vmware-tanzu/nsx-operator/pkg/apis/nsx.vmware.com/v1alpha1"
	v1alpha2 "github.com/vmware-tanzu/nsx-operator/pkg/apis/nsx.vmware.com/v1alpha2"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=nsx.vmware.com, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithResource("ipaddressallocations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().IPAddressAllocations().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("ippools"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().IPPools().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("nsxserviceaccounts"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().NSXServiceAccounts().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("networkinfos"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().NetworkInfos().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("securitypolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().SecurityPolicies().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("staticroutes"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().StaticRoutes().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("subnets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().Subnets().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("subnetports"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().SubnetPorts().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("subnetsets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().SubnetSets().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("vpcnetworkconfigurations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha1().VPCNetworkConfigurations().Informer()}, nil

		// Group=nsx.vmware.com, Version=v1alpha2
	case v1alpha2.SchemeGroupVersion.WithResource("ippools"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Nsx().V1alpha2().IPPools().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
