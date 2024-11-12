/* Copyright © 2024 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/vmware-tanzu/nsx-operator/pkg/apis/vpc/v1alpha1"
	scheme "github.com/vmware-tanzu/nsx-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SubnetConnectionBindingMapsGetter has a method to return a SubnetConnectionBindingMapInterface.
// A group's client should implement this interface.
type SubnetConnectionBindingMapsGetter interface {
	SubnetConnectionBindingMaps(namespace string) SubnetConnectionBindingMapInterface
}

// SubnetConnectionBindingMapInterface has methods to work with SubnetConnectionBindingMap resources.
type SubnetConnectionBindingMapInterface interface {
	Create(ctx context.Context, subnetConnectionBindingMap *v1alpha1.SubnetConnectionBindingMap, opts v1.CreateOptions) (*v1alpha1.SubnetConnectionBindingMap, error)
	Update(ctx context.Context, subnetConnectionBindingMap *v1alpha1.SubnetConnectionBindingMap, opts v1.UpdateOptions) (*v1alpha1.SubnetConnectionBindingMap, error)
	UpdateStatus(ctx context.Context, subnetConnectionBindingMap *v1alpha1.SubnetConnectionBindingMap, opts v1.UpdateOptions) (*v1alpha1.SubnetConnectionBindingMap, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.SubnetConnectionBindingMap, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.SubnetConnectionBindingMapList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.SubnetConnectionBindingMap, err error)
	SubnetConnectionBindingMapExpansion
}

// subnetConnectionBindingMaps implements SubnetConnectionBindingMapInterface
type subnetConnectionBindingMaps struct {
	client rest.Interface
	ns     string
}

// newSubnetConnectionBindingMaps returns a SubnetConnectionBindingMaps
func newSubnetConnectionBindingMaps(c *CrdV1alpha1Client, namespace string) *subnetConnectionBindingMaps {
	return &subnetConnectionBindingMaps{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the subnetConnectionBindingMap, and returns the corresponding subnetConnectionBindingMap object, and an error if there is any.
func (c *subnetConnectionBindingMaps) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.SubnetConnectionBindingMap, err error) {
	result = &v1alpha1.SubnetConnectionBindingMap{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("subnetconnectionbindingmaps").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SubnetConnectionBindingMaps that match those selectors.
func (c *subnetConnectionBindingMaps) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.SubnetConnectionBindingMapList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.SubnetConnectionBindingMapList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("subnetconnectionbindingmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested subnetConnectionBindingMaps.
func (c *subnetConnectionBindingMaps) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("subnetconnectionbindingmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a subnetConnectionBindingMap and creates it.  Returns the server's representation of the subnetConnectionBindingMap, and an error, if there is any.
func (c *subnetConnectionBindingMaps) Create(ctx context.Context, subnetConnectionBindingMap *v1alpha1.SubnetConnectionBindingMap, opts v1.CreateOptions) (result *v1alpha1.SubnetConnectionBindingMap, err error) {
	result = &v1alpha1.SubnetConnectionBindingMap{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("subnetconnectionbindingmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(subnetConnectionBindingMap).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a subnetConnectionBindingMap and updates it. Returns the server's representation of the subnetConnectionBindingMap, and an error, if there is any.
func (c *subnetConnectionBindingMaps) Update(ctx context.Context, subnetConnectionBindingMap *v1alpha1.SubnetConnectionBindingMap, opts v1.UpdateOptions) (result *v1alpha1.SubnetConnectionBindingMap, err error) {
	result = &v1alpha1.SubnetConnectionBindingMap{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("subnetconnectionbindingmaps").
		Name(subnetConnectionBindingMap.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(subnetConnectionBindingMap).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *subnetConnectionBindingMaps) UpdateStatus(ctx context.Context, subnetConnectionBindingMap *v1alpha1.SubnetConnectionBindingMap, opts v1.UpdateOptions) (result *v1alpha1.SubnetConnectionBindingMap, err error) {
	result = &v1alpha1.SubnetConnectionBindingMap{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("subnetconnectionbindingmaps").
		Name(subnetConnectionBindingMap.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(subnetConnectionBindingMap).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the subnetConnectionBindingMap and deletes it. Returns an error if one occurs.
func (c *subnetConnectionBindingMaps) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("subnetconnectionbindingmaps").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *subnetConnectionBindingMaps) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("subnetconnectionbindingmaps").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched subnetConnectionBindingMap.
func (c *subnetConnectionBindingMaps) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.SubnetConnectionBindingMap, err error) {
	result = &v1alpha1.SubnetConnectionBindingMap{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("subnetconnectionbindingmaps").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
