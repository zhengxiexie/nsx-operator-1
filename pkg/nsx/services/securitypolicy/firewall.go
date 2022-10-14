package securitypolicy

import (
	"sync"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/util"
)

var (
	log                        = common.Log
	MarkedForDelete            = true
	EnforceRevisionCheckParam  = false
	Converter                  = common.Converter
	ResourceTypeSecurityPolicy = common.ResourceTypeSecurityPolicy
	ResourceTypeRule           = common.ResourceTypeRule
	ResourceTypeGroup          = common.ResourceTypeGroup
	operateSecurityPolicyStore common.OperateStore
	operateGroupStore          common.OperateStore
	operateRuleStore           common.OperateStore
	securityPolicyStore        cache.Indexer
	ruleStore                  cache.Indexer
	groupStore                 cache.Indexer
)

type SecurityPolicyService struct {
	common.Service
}

// InitializeSecurityPolicy sync NSX resources
func InitializeSecurityPolicy(service common.Service) (*SecurityPolicyService, error) {
	wg := sync.WaitGroup{}
	wgDone := make(chan bool)
	fatalErrors := make(chan error)

	wg.Add(3)

	securityService := &SecurityPolicyService{Service: service}
	operateSecurityPolicyStore = &OperateSecurityPolicyStore{securityService}
	operateGroupStore = &OperateGroupStore{securityService}
	operateRuleStore = &OperateRuleStore{securityService}

	InitializeStore(securityService)
	securityPolicyStore = securityService.Store[ResourceTypeSecurityPolicy]
	ruleStore = securityService.Store[ResourceTypeRule]
	groupStore = securityService.Store[ResourceTypeGroup]

	go securityService.QueryResource(&wg, fatalErrors, ResourceTypeSecurityPolicy, operateSecurityPolicyStore)
	go securityService.QueryResource(&wg, fatalErrors, ResourceTypeGroup, operateGroupStore)
	go securityService.QueryResource(&wg, fatalErrors, ResourceTypeRule, operateRuleStore)

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		break
	case err := <-fatalErrors:
		close(fatalErrors)
		return securityService, err
	}

	return securityService, nil
}

func (service *SecurityPolicyService) CreateOrUpdateSecurityPolicy(obj *v1alpha1.SecurityPolicy) error {
	nsxSecurityPolicy, nsxGroups, err := service.buildSecurityPolicy(obj)
	if err != nil {
		log.Error(err, "failed to build SecurityPolicy")
		return err
	}

	if len(nsxSecurityPolicy.Scope) == 0 {
		log.Info("SecurityPolicy has empty policy-level appliedTo")
	}

	existingSecurityPolicy := model.SecurityPolicy{}
	res, exists, err := securityPolicyStore.GetByKey(*nsxSecurityPolicy.Id)
	if err != nil {
		log.Error(err, "failed to get security policy", "SecurityPolicy", nsxSecurityPolicy)
	} else if exists {
		existingSecurityPolicy = res.(model.SecurityPolicy)
	}

	indexResults, err := ruleStore.ByIndex(util.TagScopeSecurityPolicyCRUID, string(obj.UID))
	if err != nil {
		log.Error(err, "failed to get rules by security policy UID", "SecurityPolicyCR.UID", obj.UID)
		return err
	}
	existingRules := make([]model.Rule, 0)
	for _, rule := range indexResults {
		existingRules = append(existingRules, rule.(model.Rule))
	}

	indexResults, err = groupStore.ByIndex(util.TagScopeSecurityPolicyCRUID, string(obj.UID))
	if err != nil {
		log.Error(err, "failed to get groups by security policy UID", "SecurityPolicyCR.UID", obj.UID)
		return err
	}
	existingGroups := make([]model.Group, 0)
	for _, group := range indexResults {
		existingGroups = append(existingGroups, group.(model.Group))
	}

	changedSecurityPolicy := service.securityPolicyCompare(&existingSecurityPolicy, nsxSecurityPolicy)
	changedRules, staleRules := service.rulesCompare(existingRules, nsxSecurityPolicy.Rules)
	changedGroups, staleGroups := service.groupsCompare(existingGroups, *nsxGroups)

	if changedSecurityPolicy == nil && len(changedRules) == 0 && len(staleRules) == 0 && len(changedGroups) == 0 && len(staleGroups) == 0 {
		log.Info("security policy, rules and groups are not changed, skip updating them", "nsxSecurityPolicy.Id", nsxSecurityPolicy.Id)
		return nil
	}

	var finalSecurityPolicy *model.SecurityPolicy
	if changedSecurityPolicy == nil {
		finalSecurityPolicy = &existingSecurityPolicy
	} else {
		finalSecurityPolicy = changedSecurityPolicy
	}

	finalRules := make([]model.Rule, 0)
	for i := len(staleRules) - 1; i >= 0; i-- { // Don't use range, it would copy the element
		staleRules[i].MarkedForDelete = &MarkedForDelete // InfraClient need this field to delete the group
	}
	finalRules = append(finalRules, staleRules...)
	finalRules = append(finalRules, changedRules...)
	finalSecurityPolicy.Rules = finalRules

	finalGroups := make([]model.Group, 0)
	for i := len(staleGroups) - 1; i >= 0; i-- { // Don't use range, it would copy the element
		staleGroups[i].MarkedForDelete = &MarkedForDelete // InfraClient need this field to delete the group
	}
	finalGroups = append(finalGroups, staleGroups...)
	finalGroups = append(finalGroups, changedGroups...)

	// WrapHighLevelSecurityPolicy will modify the input security policy, so we need to make a copy for the following store update.
	finalSecurityPolicyCopy := *finalSecurityPolicy
	finalSecurityPolicyCopy.Rules = finalSecurityPolicy.Rules
	infraSecurityPolicy, error := service.WrapHierarchySecurityPolicy(finalSecurityPolicy, finalGroups)
	if error != nil {
		return error
	}
	err = service.NSXClient.InfraClient.Patch(*infraSecurityPolicy, &EnforceRevisionCheckParam)
	if err != nil {
		return err
	}

	// The steps below know how to deal with CR, if there is MarkedForDelete, then delete it from store,
	// otherwise add or update it to store.
	if changedSecurityPolicy != nil {
		err = operateSecurityPolicyStore.CRUDResource(&finalSecurityPolicyCopy)
		if err != nil {
			return err
		}
	}
	if !(len(changedRules) == 0 && len(staleRules) == 0) {
		err = operateRuleStore.CRUDResource(&finalSecurityPolicyCopy)
		if err != nil {
			return err
		}
	}
	if !(len(changedGroups) == 0 && len(staleGroups) == 0) {
		err = operateGroupStore.CRUDResource(&finalGroups)
		if err != nil {
			return err
		}
	}
	log.Info("successfully created or updated nsxSecurityPolicy", "nsxSecurityPolicy", finalSecurityPolicyCopy)
	return nil
}

func (service *SecurityPolicyService) DeleteSecurityPolicy(obj interface{}) error {
	var nsxSecurityPolicy *model.SecurityPolicy
	g := make([]model.Group, 0)
	nsxGroups := &g
	switch sp := obj.(type) {
	case *v1alpha1.SecurityPolicy:
		var err error
		nsxSecurityPolicy, nsxGroups, err = service.buildSecurityPolicy(sp)
		if err != nil {
			log.Error(err, "failed to build SecurityPolicy")
			return err
		}
	case types.UID:
		indexResults, err := securityPolicyStore.ByIndex(util.TagScopeSecurityPolicyCRUID, string(sp))
		if err != nil {
			log.Error(err, "failed to get security policy", "UID", string(sp))
			return err
		}
		if len(indexResults) == 0 {
			log.Info("did not get security policy with index", "UID", string(sp))
			return nil
		}
		t := indexResults[0].(model.SecurityPolicy)
		nsxSecurityPolicy = &t

		indexResults, err = groupStore.ByIndex(util.TagScopeSecurityPolicyCRUID, string(sp))
		if err != nil {
			log.Error(err, "failed to get groups", "UID", string(sp))
			return err
		}
		if len(indexResults) == 0 {
			log.Info("did not get groups with index", "UID", string(sp))
		}
		for _, group := range indexResults {
			*nsxGroups = append(*nsxGroups, group.(model.Group))
		}
	}

	nsxSecurityPolicy.MarkedForDelete = &MarkedForDelete
	for i := len(*nsxGroups) - 1; i >= 0; i-- { // Don't use range, it would copy the element
		(*nsxGroups)[i].MarkedForDelete = &MarkedForDelete
	}
	for i := len(nsxSecurityPolicy.Rules) - 1; i >= 0; i-- { // Don't use range, it would copy the element
		nsxSecurityPolicy.Rules[i].MarkedForDelete = &MarkedForDelete
	}

	// WrapHighLevelSecurityPolicy will modify the input security policy, so we need to make a copy for the following store update.
	finalSecurityPolicyCopy := *nsxSecurityPolicy
	finalSecurityPolicyCopy.Rules = nsxSecurityPolicy.Rules
	infraSecurityPolicy, error := service.WrapHierarchySecurityPolicy(nsxSecurityPolicy, *nsxGroups)
	if error != nil {
		return error
	}
	err := service.NSXClient.InfraClient.Patch(*infraSecurityPolicy, &EnforceRevisionCheckParam)
	if err != nil {
		return err
	}
	err = operateSecurityPolicyStore.CRUDResource(nsxSecurityPolicy)
	if err != nil {
		return err
	}
	err = operateGroupStore.CRUDResource(nsxGroups)
	if err != nil {
		return err
	}
	err = operateRuleStore.CRUDResource(&finalSecurityPolicyCopy)
	if err != nil {
		return err
	}
	log.Info("successfully deleted  nsxSecurityPolicy", "nsxSecurityPolicy", nsxSecurityPolicy)
	return nil
}

func (service *SecurityPolicyService) createOrUpdateGroups(nsxGroups []model.Group) error {
	for _, group := range nsxGroups {
		err := service.NSXClient.GroupClient.Patch(getDomain(service), *group.Id, group)
		if err != nil {
			return err
		}
		err = groupStore.Add(group)
		log.V(2).Info("add group to store", "group", group.Id)
		if err != nil {
			return err
		}
	}
	log.Info("successfully create or update group", "groups", nsxGroups)
	return nil
}

func (service *SecurityPolicyService) ListSecurityPolicyID() sets.String {
	groups := groupStore.ListIndexFuncValues(util.TagScopeSecurityPolicyCRUID)
	groupSet := sets.NewString()
	for _, group := range groups {
		groupSet.Insert(group)
	}
	securityPolicies := securityPolicyStore.ListIndexFuncValues(util.TagScopeSecurityPolicyCRUID)
	policySet := sets.NewString()
	for _, policy := range securityPolicies {
		policySet.Insert(policy)
	}
	return groupSet.Union(policySet)
}
