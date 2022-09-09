/* Copyright © 2021 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package controllers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/nsx-operator/pkg/metrics"
	_ "github.com/vmware-tanzu/nsx-operator/pkg/nsx/ratelimiter"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services"
	"github.com/vmware-tanzu/nsx-operator/pkg/util"
)

var (
	log                     = logf.Log.WithName("controller").WithName("securitypolicy")
	resultRequeueAfter5mins = ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Minute}
)

const (
	WCP_SYSTEM_RESOURCE = "vmware-system-shared-t1"
	METRIC_RES_TYPE     = "securitypolicy"
)

// SecurityPolicyReconciler reconciles a SecurityPolicy object
type SecurityPolicyReconciler struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	Service *services.SecurityPolicyService
}

func updateFail(r *SecurityPolicyReconciler, c *context.Context, o *v1alpha1.SecurityPolicy, e *error) {
	r.setSecurityPolicyReadyStatusFalse(c, o, e)
	metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerUpdateFailTotal, METRIC_RES_TYPE)
}

func (r *SecurityPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &v1alpha1.SecurityPolicy{}
	log.Info("reconciling securitypolicy CR", "securitypolicy", req.NamespacedName)
	metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerSyncTotal, METRIC_RES_TYPE)

	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		log.Error(err, "unable to fetch security policy CR", "req", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Since securitypolicy service can only be activated from nsx 3.2.0 onwards,
	// So need to check nsx version before starting securitypolicy Reconcile
	if !r.Service.NSXClient.NSXCheckVersion() {
		err := errors.New("NSX version check failed")
		log.Error(err, "SecurityPolicy feature is not supported")
		updateFail(r, &ctx, obj, &err)
		return resultRequeueAfter5mins, nil
	}

	if obj.ObjectMeta.DeletionTimestamp.IsZero() {
		metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerUpdateTotal, METRIC_RES_TYPE)
		if !controllerutil.ContainsFinalizer(obj, util.FinalizerName) {
			controllerutil.AddFinalizer(obj, util.FinalizerName)
			if err := r.Client.Update(ctx, obj); err != nil {
				metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerUpdateFailTotal, METRIC_RES_TYPE)
				return ctrl.Result{}, err
			}
			log.V(1).Info("added finalizer on securitypolicy CR", "securitypolicy", req.NamespacedName)
		}

		if isCRInSysNs, err := r.isCRRequestedInSystemNamespace(&ctx, &req); err != nil {
			metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerUpdateFailTotal, METRIC_RES_TYPE)
			return ctrl.Result{}, err
		} else if isCRInSysNs {
			err = errors.New("Security Policy CR cannot be created in System Namespace")
			log.Error(err, "failed to create security policy CR", "securitypolicy", req.NamespacedName)
			metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerUpdateFailTotal, METRIC_RES_TYPE)
			return ctrl.Result{}, err
		}

		if err := r.Service.CreateOrUpdateSecurityPolicy(obj); err != nil {
			log.Error(err, "failed to create or update security policy CR", "securitypolicy", req.NamespacedName)
			r.setSecurityPolicyReadyStatusFalse(&ctx, obj, &err)
			metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerUpdateFailTotal, METRIC_RES_TYPE)
			return ctrl.Result{}, err
		}
		r.setSecurityPolicyReadyStatusTrue(&ctx, obj)
		metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerUpdateSuccessTotal, METRIC_RES_TYPE)
	} else {
		if containsString(obj.GetFinalizers(), util.FinalizerName) {
			metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerDeleteTotal, METRIC_RES_TYPE)
			if err := r.Service.DeleteSecurityPolicy(obj); err != nil {
				metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerDeleteFailTotal, METRIC_RES_TYPE)
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(obj, util.FinalizerName)
			if err := r.Client.Update(ctx, obj); err != nil {
				metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerDeleteFailTotal, METRIC_RES_TYPE)
				return ctrl.Result{}, err
			}
			log.V(1).Info("removed finalizer on securitypolicy CR", "securitypolicy", req.NamespacedName)
			metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerDeleteSuccessTotal, METRIC_RES_TYPE)
		} else {
			// only print a message because it's not a normal case
			log.Info("securitypolicy CR is being deleted but its finalizers cannot be recognized", "securitypolicy", req.NamespacedName)
		}
	}

	return ctrl.Result{}, nil
}

func (r *SecurityPolicyReconciler) isCRRequestedInSystemNamespace(ctx *context.Context, req *ctrl.Request) (bool, error) {
	nsObj := &v1.Namespace{}

	if err := r.Client.Get(*ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Namespace}, nsObj); err != nil {
		log.Error(err, "unable to fetch namespace associated with security policy CR", "req", req.NamespacedName)
		return false, client.IgnoreNotFound(err)
	}

	if isSysNs, ok := nsObj.Annotations[WCP_SYSTEM_RESOURCE]; ok && strings.ToLower(isSysNs) == "true" {
		return true, nil
	}

	return false, nil
}

func (r *SecurityPolicyReconciler) setSecurityPolicyReadyStatusTrue(ctx *context.Context, sec_policy *v1alpha1.SecurityPolicy) {
	newConditions := []v1alpha1.SecurityPolicyCondition{
		{
			Type:    v1alpha1.SecurityPolicyReady,
			Status:  v1.ConditionTrue,
			Message: "NSX Security Policy has been successfully created/updated",
			Reason:  "NSX API returned 200 response code for PATCH",
		},
	}
	r.updateSecurityPolicyStatusConditions(ctx, sec_policy, newConditions)
}

func (r *SecurityPolicyReconciler) setSecurityPolicyReadyStatusFalse(ctx *context.Context, sec_policy *v1alpha1.SecurityPolicy, err *error) {
	newConditions := []v1alpha1.SecurityPolicyCondition{
		{
			Type:    v1alpha1.SecurityPolicyReady,
			Status:  v1.ConditionFalse,
			Message: "NSX Security Policy could not be created/updated",
			Reason:  fmt.Sprintf("Error occurred while processing the Security Policy CR. Please check the config and try again. Error: %v", *err),
		},
	}
	r.updateSecurityPolicyStatusConditions(ctx, sec_policy, newConditions)
}

func (r *SecurityPolicyReconciler) updateSecurityPolicyStatusConditions(ctx *context.Context, sec_policy *v1alpha1.SecurityPolicy, newConditions []v1alpha1.SecurityPolicyCondition) {
	conditionsUpdated := false
	for i := range newConditions {
		if r.mergeSecurityPolicyStatusCondition(ctx, sec_policy, &newConditions[i]) {
			conditionsUpdated = true
		}
	}
	if conditionsUpdated {
		r.Client.Status().Update(*ctx, sec_policy)
		log.V(1).Info("Updated Security Policy CRD", "Name", sec_policy.Name, "Namespace", sec_policy.Namespace, "New Conditions", newConditions)
	}
}

func (r *SecurityPolicyReconciler) mergeSecurityPolicyStatusCondition(ctx *context.Context, sec_policy *v1alpha1.SecurityPolicy, newCondition *v1alpha1.SecurityPolicyCondition) bool {
	matchedCondition := getExistingConditionOfType(newCondition.Type, sec_policy.Status.Conditions)

	if reflect.DeepEqual(matchedCondition, newCondition) {
		log.V(2).Info("Conditions already match", "New Condition", newCondition, "Existing Condition", matchedCondition)
		return false
	}

	if matchedCondition != nil {
		matchedCondition.Reason = newCondition.Reason
		matchedCondition.Message = newCondition.Message
		matchedCondition.Status = newCondition.Status
	} else {
		sec_policy.Status.Conditions = append(sec_policy.Status.Conditions, *newCondition)
	}
	return true
}

func getExistingConditionOfType(conditionType v1alpha1.SecurityPolicyStatusCondition, existingConditions []v1alpha1.SecurityPolicyCondition) *v1alpha1.SecurityPolicyCondition {
	for i := range existingConditions {
		if existingConditions[i].Type == conditionType {
			return &existingConditions[i]
		}
	}
	return nil
}

func containsString(source []string, target string) bool {
	for _, item := range source {
		if item == target {
			return true
		}
	}
	return false
}

func (r *SecurityPolicyReconciler) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SecurityPolicy{}).
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
		Complete(r)
}

// Start setup manager and launch GC
func (r *SecurityPolicyReconciler) Start(mgr ctrl.Manager) error {
	err := r.setupWithManager(mgr)
	if err != nil {
		return err
	}

	go r.GarbageCollector(make(chan bool), util.GCInterval)
	return nil
}

// GarbageCollector collect securitypolicy which has been removed from crd.
// cancel is used to break the loop during UT
func (r *SecurityPolicyReconciler) GarbageCollector(cancel chan bool, timeout time.Duration) {
	ctx := context.Background()
	for {
		select {
		case <-cancel:
			return
		case <-time.After(timeout):
		}

		keys := r.Service.ListSecurityPolicyKeys()
		if len(keys) == 0 {
			continue
		}

		policyMap := make(map[string]v1alpha1.SecurityPolicy)
		policyList := &v1alpha1.SecurityPolicyList{}
		err := r.Client.List(ctx, policyList)
		if err != nil {
			log.Error(err, "failed to list security policy CR")
			continue
		}

		for _, policy := range policyList.Items {
			policyMap[string(policy.UID)] = policy
		}

		for _, key := range keys {
			t, exists, err := r.Service.SecurityPolicyStore.GetByKey(key)
			if err != nil || !exists {
				log.Error(err, "failed to get security policy from store")
				continue
			}
			nsxPolicy := t.(model.SecurityPolicy)
			for _, tag := range nsxPolicy.Tags {
				if *tag.Scope == util.TagScopeSecurityPolicyCRUID {
					nsxPolicyUID := *tag.Tag
					if policy, ok := policyMap[nsxPolicyUID]; !ok {
						log.V(1).Info("GC collected SecurityPolicy CR", "policy", policy)
						metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerDeleteTotal, METRIC_RES_TYPE)
						err = r.Service.DeleteSecurityPolicy(&nsxPolicy)
						if err != nil {
							metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerDeleteFailTotal, METRIC_RES_TYPE)
						} else {
							metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerDeleteSuccessTotal, METRIC_RES_TYPE)
						}
					}
				}
			}
		}
	}
}
