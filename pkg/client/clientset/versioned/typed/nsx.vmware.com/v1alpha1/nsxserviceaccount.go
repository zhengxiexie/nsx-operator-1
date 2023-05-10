/* Copyright © 2022 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/vmware-tanzu/nsx-operator/pkg/apis/nsx.vmware.com/v1alpha1"
	scheme "github.com/vmware-tanzu/nsx-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NSXServiceAccountsGetter has a method to return a NSXServiceAccountInterface.
// A group's client should implement this interface.
type NSXServiceAccountsGetter interface {
	NSXServiceAccounts(namespace string) NSXServiceAccountInterface
}

// NSXServiceAccountInterface has methods to work with NSXServiceAccount resources.
type NSXServiceAccountInterface interface {
	Create(ctx context.Context, nSXServiceAccount *v1alpha1.NSXServiceAccount, opts v1.CreateOptions) (*v1alpha1.NSXServiceAccount, error)
	Update(ctx context.Context, nSXServiceAccount *v1alpha1.NSXServiceAccount, opts v1.UpdateOptions) (*v1alpha1.NSXServiceAccount, error)
	UpdateStatus(ctx context.Context, nSXServiceAccount *v1alpha1.NSXServiceAccount, opts v1.UpdateOptions) (*v1alpha1.NSXServiceAccount, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.NSXServiceAccount, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.NSXServiceAccountList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NSXServiceAccount, err error)
	NSXServiceAccountExpansion
}

// nSXServiceAccounts implements NSXServiceAccountInterface
type nSXServiceAccounts struct {
	client rest.Interface
	ns     string
}

// newNSXServiceAccounts returns a NSXServiceAccounts
func newNSXServiceAccounts(c *NsxV1alpha1Client, namespace string) *nSXServiceAccounts {
	return &nSXServiceAccounts{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the nSXServiceAccount, and returns the corresponding nSXServiceAccount object, and an error if there is any.
func (c *nSXServiceAccounts) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.NSXServiceAccount, err error) {
	result = &v1alpha1.NSXServiceAccount{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("nsxserviceaccounts").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NSXServiceAccounts that match those selectors.
func (c *nSXServiceAccounts) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NSXServiceAccountList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.NSXServiceAccountList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("nsxserviceaccounts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested nSXServiceAccounts.
func (c *nSXServiceAccounts) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("nsxserviceaccounts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a nSXServiceAccount and creates it.  Returns the server's representation of the nSXServiceAccount, and an error, if there is any.
func (c *nSXServiceAccounts) Create(ctx context.Context, nSXServiceAccount *v1alpha1.NSXServiceAccount, opts v1.CreateOptions) (result *v1alpha1.NSXServiceAccount, err error) {
	result = &v1alpha1.NSXServiceAccount{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("nsxserviceaccounts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(nSXServiceAccount).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a nSXServiceAccount and updates it. Returns the server's representation of the nSXServiceAccount, and an error, if there is any.
func (c *nSXServiceAccounts) Update(ctx context.Context, nSXServiceAccount *v1alpha1.NSXServiceAccount, opts v1.UpdateOptions) (result *v1alpha1.NSXServiceAccount, err error) {
	result = &v1alpha1.NSXServiceAccount{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("nsxserviceaccounts").
		Name(nSXServiceAccount.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(nSXServiceAccount).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *nSXServiceAccounts) UpdateStatus(ctx context.Context, nSXServiceAccount *v1alpha1.NSXServiceAccount, opts v1.UpdateOptions) (result *v1alpha1.NSXServiceAccount, err error) {
	result = &v1alpha1.NSXServiceAccount{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("nsxserviceaccounts").
		Name(nSXServiceAccount.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(nSXServiceAccount).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the nSXServiceAccount and deletes it. Returns an error if one occurs.
func (c *nSXServiceAccounts) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("nsxserviceaccounts").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *nSXServiceAccounts) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("nsxserviceaccounts").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched nSXServiceAccount.
func (c *nSXServiceAccounts) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NSXServiceAccount, err error) {
	result = &v1alpha1.NSXServiceAccount{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("nsxserviceaccounts").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
