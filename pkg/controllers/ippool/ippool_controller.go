/* Copyright © 2021 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package ippool

import (
	"context"
	"runtime"
	"time"

	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha2"
	"github.com/vmware-tanzu/nsx-operator/pkg/controllers/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
	"github.com/vmware-tanzu/nsx-operator/pkg/metrics"
	servicecommon "github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/ippool"
)

var (
	log           = logger.Log
	resultNormal  = common.ResultNormal
	resultRequeue = common.ResultRequeue
	MetricResType = common.MetricResTypeSecurityPolicy
)

// Reconciler reconciles a IPPool object
type Reconciler struct {
	client.Client
	Scheme  *apimachineryruntime.Scheme
	Service *ippool.IPPoolService
}

func containsString(source []string, target string) bool {
	for _, item := range source {
		if item == target {
			return true
		}
	}
	return false
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &v1alpha2.IPPool{}
	log.Info("reconciling ippool CR", "ippool", req.NamespacedName)
	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		log.Error(err, "unable to fetch ippool CR", "req", req.NamespacedName)
		return resultNormal, client.IgnoreNotFound(err)
	}
	if obj.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(obj, servicecommon.FinalizerName) {
			controllerutil.AddFinalizer(obj, servicecommon.FinalizerName)
			if err := r.Client.Update(ctx, obj); err != nil {
				return resultRequeue, err
			}
			log.V(1).Info("added finalizer on ippool CR", "ippool", req.NamespacedName)
		}

		subnetCidrUpdated, ipPoolSubnetsUpdated, err := r.Service.CreateOrUpdateIPPool(obj)
		if err != nil {
			log.Error(err, "operate failed, would retry exponentially", "ippool", req.NamespacedName)
			return resultRequeue, err
		}
		if !r.Service.FullyRealized(obj) {
			if subnetCidrUpdated || ipPoolSubnetsUpdated {
				err = r.Client.Status().Update(ctx, obj)
				if err != nil {
					return resultRequeue, err
				}
			}
			log.Info("put back ippool again, some subnets unrealized", "subnets", r.Service.GetUnrealizedSubnetNames(obj))
			return resultRequeue, nil
		} else {
			if subnetCidrUpdated || ipPoolSubnetsUpdated {
				err = r.Client.Status().Update(ctx, obj)
				if err != nil {
					return resultRequeue, err
				}
				log.Info("successfully reconcile ippool CR", "ippool", obj)
			} else {
				log.Info("full realized already, and resources are not changed, skip updating them", "obj", obj)
			}
		}
	} else {
		if containsString(obj.GetFinalizers(), servicecommon.FinalizerName) {
			metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerDeleteTotal, MetricResType)
			if err := r.Service.DeleteIPPool(obj); err != nil {
				return resultRequeue, err
			}
			controllerutil.RemoveFinalizer(obj, servicecommon.FinalizerName)
			if err := r.Client.Update(ctx, obj); err != nil {
				return resultRequeue, err
			}
			log.V(1).Info("removed finalizer on ippool CR", "ippool", req.NamespacedName)
		} else {
			// only print a message because it's not a normal case
			log.Info("ippool CR is being deleted but its finalizers cannot be recognized", "ippool", req.NamespacedName)
		}
		log.Info("successfully deleted ippool CR", "ippool", obj)
	}
	return resultNormal, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.IPPool{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Ignore updates to CR status in which case metadata.Generation does not change
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				// Suppress Delete events to avoid filtering them out in the Reconcile function
				return false
			},
		}).
		WithOptions(
			controller.Options{
				MaxConcurrentReconciles: runtime.NumCPU(),
			}).
		Complete(r)
}

// Start setup manager and launch GC
func (r *Reconciler) Start(mgr ctrl.Manager) error {
	err := r.SetupWithManager(mgr)
	if err != nil {
		return err
	}
	go r.IPPoolGarbageCollector(make(chan bool), servicecommon.GCInterval)
	return nil
}

// IPPoolGarbageCollector collect ippool which has been removed from crd.
// cancel is used to break the loop during UT
func (r *Reconciler) IPPoolGarbageCollector(cancel chan bool, timeout time.Duration) {
	ctx := context.Background()
	log.Info("ippool garbage collector started")
	for {
		select {
		case <-cancel:
			return
		case <-time.After(timeout):
		}
		nsxIPPoolSet := r.Service.ListIPPoolID()
		if len(nsxIPPoolSet) == 0 {
			continue
		}
		ipPoolList := &v1alpha2.IPPoolList{}
		err := r.Client.List(ctx, ipPoolList)
		if err != nil {
			log.Error(err, "failed to list ip pool CR")
			continue
		}

		CRIPPoolSet := sets.NewString()
		for _, ipp := range ipPoolList.Items {
			CRIPPoolSet.Insert(string(ipp.UID))
		}

		log.V(2).Info("ippool garbage collector", "nsxIPPoolSet", nsxIPPoolSet, "CRIPPoolSet", CRIPPoolSet)

		for elem := range nsxIPPoolSet {
			if CRIPPoolSet.Has(elem) {
				continue
			}
			log.Info("GC collected ip pool CR", "UID", elem)
			err = r.Service.DeleteIPPool(types.UID(elem))
			if err != nil {
				log.Error(err, "failed to delete ip pool CR", "UID", elem)
			}
		}
	}
}
