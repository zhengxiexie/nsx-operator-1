package services

import (
	"github.com/vmware/vsphere-automation-sdk-go/runtime/bindings"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"k8s.io/client-go/tools/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/nsx-operator/pkg/config"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx"
	"github.com/vmware-tanzu/nsx-operator/pkg/util"
)

var (
	log                       = logf.Log.WithName("service").WithName("firewall")
	MarkedForDelete           = true
	EnforceRevisionCheckParam = false
	Converter                 *bindings.TypeConverter
)

func init() {
	Converter = bindings.NewTypeConverter()
	Converter.SetMode(bindings.REST)
}

type SecurityPolicyService struct {
	NSXClient           *nsx.Client
	NSXConfig           *config.NSXOperatorConfig
	GroupStore          cache.Indexer
	SecurityPolicyStore cache.Indexer
	RuleStore           cache.Indexer
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
	res, exists, err := service.SecurityPolicyStore.GetByKey(*nsxSecurityPolicy.Id)
	if err != nil {
		log.Error(err, "failed to get security policy", "SecurityPolicy", nsxSecurityPolicy)
	} else if exists {
		existingSecurityPolicy = res.(model.SecurityPolicy)
	}

	indexResults, err := service.RuleStore.ByIndex(util.TagScopeSecurityPolicyCRUID, string(obj.UID))
	if err != nil {
		log.Error(err, "failed to get rules by security policy UID", "SecurityPolicyCR.UID", obj.UID)
		return err
	}
	existingRules := make([]model.Rule, 0)
	for _, rule := range indexResults {
		existingRules = append(existingRules, rule.(model.Rule))
	}

	indexResults, err = service.GroupStore.ByIndex(util.TagScopeSecurityPolicyCRUID, string(obj.UID))
	if err != nil {
		log.Error(err, "failed to get groups by security policy UID", "SecurityPolicyCR.UID", obj.UID)
		return err
	}
	existingGroups := make([]model.Group, 0)
	for _, group := range indexResults {
		existingGroups = append(existingGroups, group.(model.Group))
	}

	changedSecurityPolicy := service.securityPolicyEqual(&existingSecurityPolicy, nsxSecurityPolicy)
	changedRules, staleRules := service.rulesEqual(existingRules, nsxSecurityPolicy.Rules)
	changedGroups, staleGroups := service.groupsEqual(existingGroups, *nsxGroups)

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
	for i := len(staleRules) - 1; i >= 0; i-- { // Don't use range, it would copy the element
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

	// The method Operate*CR* knows how to deal with CR, if there is MarkedForDelete, then delete it from store,
	// otherwise add or update it to store.
	if changedSecurityPolicy != nil {
		err = service.OperateSecurityStore(&finalSecurityPolicyCopy)
		if err != nil {
			return err
		}
	}
	if !(len(changedRules) == 0 && len(staleRules) == 0) {
		err = service.OperateRulesStore(&finalSecurityPolicyCopy)
		if err != nil {
			return err
		}
	}
	if !(len(changedGroups) == 0 && len(staleGroups) == 0) {
		err = service.OperateGroupsStore(&finalGroups)
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
	switch o := obj.(type) {
	case *v1alpha1.SecurityPolicy:
		var err error
		nsxSecurityPolicy, nsxGroups, err = service.buildSecurityPolicy(o)
		if err != nil {
			log.Error(err, "failed to build SecurityPolicy")
			return err
		}
	case *model.SecurityPolicy:
		// We can delete the security policy directly by its ID,
		// It's related resources will be deleted automatically,
		// So don't worry we have no nsxGroups and nsxRules.
		nsxSecurityPolicy = o
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
	enforceRevisionCheckParam := false
	err := service.NSXClient.InfraClient.Patch(*infraSecurityPolicy, &enforceRevisionCheckParam)
	if err != nil {
		return err
	}
	err = service.OperateSecurityStore(nsxSecurityPolicy)
	if err != nil {
		return err
	}
	err = service.OperateGroupsStore(nsxGroups)
	if err != nil {
		return err
	}
	err = service.OperateRulesStore(&finalSecurityPolicyCopy)
	if err != nil {
		return err
	}
	log.Info("successfully deleted  nsxSecurityPolicy", "nsxSecurityPolicy", nsxSecurityPolicy)
	return nil
}

func (service *SecurityPolicyService) ListSecurityPolicyKeys() []string {
	return service.SecurityPolicyStore.ListKeys()
}
