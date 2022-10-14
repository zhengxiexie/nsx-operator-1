package securitypolicy

import (
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"k8s.io/client-go/tools/cache"

	"github.com/vmware-tanzu/nsx-operator/pkg/util"
)

var key = util.TagScopeSecurityPolicyCRUID

func keyFuncSecurityPolicy(obj interface{}) (string, error) {
	o := obj.(model.SecurityPolicy)
	return *o.Id, nil
}

func keyFuncGroup(obj interface{}) (string, error) {
	o := obj.(model.Group)
	return *o.Id, nil
}

func keyFuncRule(obj interface{}) (string, error) {
	o := obj.(model.Rule)
	return *o.Id, nil
}

func indexFuncSecurityPolicy(obj interface{}) ([]string, error) {
	o := obj.(model.SecurityPolicy)
	return filterTag(o.Tags), nil
}

func indexFuncGroup(obj interface{}) ([]string, error) {
	o := obj.(model.Group)
	return filterTag(o.Tags), nil
}

func indexFuncRule(obj interface{}) ([]string, error) {
	o := obj.(model.Rule)
	return filterTag(o.Tags), nil
}

var filterTag = func(v []model.Tag) []string {
	res := make([]string, 0, 5)
	for _, tag := range v {
		if *tag.Scope == key {
			res = append(res, *tag.Tag)
		}
	}
	return res
}

func InitializeStore(service *SecurityPolicyService) {
	service.Store = make(map[string]cache.Indexer)
	service.Store[ResourceTypeSecurityPolicy] = cache.NewIndexer(keyFuncSecurityPolicy, cache.Indexers{key: indexFuncSecurityPolicy})
	service.Store[ResourceTypeGroup] = cache.NewIndexer(keyFuncGroup, cache.Indexers{key: indexFuncGroup})
	service.Store[ResourceTypeRule] = cache.NewIndexer(keyFuncRule, cache.Indexers{key: indexFuncRule})
}

type OperateSecurityPolicyStore struct {
	securityService *SecurityPolicyService
}

type OperateRuleStore struct {
	securityService *SecurityPolicyService
}

type OperateGroupStore struct {
	securityService *SecurityPolicyService
}

func (o *OperateSecurityPolicyStore) TransResourceToStore(entity *data.StructValue) error {
	obj, err := Converter.ConvertToGolang(entity, model.SecurityPolicyBindingType())
	if err != nil {
		for _, e := range err {
			return e
		}
	}
	sp, _ := obj.(model.SecurityPolicy)
	err2 := o.securityService.Store[ResourceTypeSecurityPolicy].Add(sp)
	if err2 != nil {
		return err2
	}
	return nil
}

func (o *OperateRuleStore) TransResourceToStore(entity *data.StructValue) error {
	obj, err := Converter.ConvertToGolang(entity, model.RuleBindingType())
	if err != nil {
		for _, e := range err {
			return e
		}
	}
	rule, _ := obj.(model.Rule)
	err2 := o.securityService.Store[ResourceTypeRule].Add(rule)
	if err2 != nil {
		return err2
	}
	return nil
}

func (o *OperateGroupStore) TransResourceToStore(entity *data.StructValue) error {
	obj, err := Converter.ConvertToGolang(entity, model.GroupBindingType())
	if err != nil {
		for _, e := range err {
			return e
		}
	}
	group, _ := obj.(model.Group)
	err2 := o.securityService.Store[ResourceTypeGroup].Add(group)
	if err2 != nil {
		return err2
	}
	return nil
}

func (o *OperateSecurityPolicyStore) CRUDResource(i interface{}) error {
	if i == nil {
		return nil
	}
	sp := i.(*model.SecurityPolicy)
	if sp.MarkedForDelete != nil && *sp.MarkedForDelete {
		err := o.securityService.Store[ResourceTypeSecurityPolicy].Delete(*sp) // Pass in the object to be deleted, not the pointer
		log.V(1).Info("delete security policy from store", "securitypolicy", sp)
		if err != nil {
			return err
		}
	} else {
		err := o.securityService.Store[ResourceTypeSecurityPolicy].Add(*sp)
		log.V(1).Info("add security policy to store", "securitypolicy", sp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *OperateRuleStore) CRUDResource(i interface{}) error {
	sp := i.(*model.SecurityPolicy)
	for _, rule := range sp.Rules {
		if rule.MarkedForDelete != nil && *rule.MarkedForDelete {
			err := o.securityService.Store[ResourceTypeRule].Delete(rule)
			log.V(1).Info("delete rule from store", "rule", rule)
			if err != nil {
				return err
			}
		} else {
			err := o.securityService.Store[ResourceTypeRule].Add(rule)
			log.V(1).Info("add rule to store", "rule", rule)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *OperateGroupStore) CRUDResource(i interface{}) error {
	gs := i.(*[]model.Group)
	for _, group := range *gs {
		if group.MarkedForDelete != nil && *group.MarkedForDelete {
			err := o.securityService.Store[ResourceTypeGroup].Delete(group)
			log.V(1).Info("delete group from store", "group", group)
			if err != nil {
				return err
			}
		} else {
			err := o.securityService.Store[ResourceTypeGroup].Add(group)
			log.V(1).Info("add group to store", "group", group)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
