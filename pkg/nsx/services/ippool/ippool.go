package ippool

import (
	"sync"
	"time"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha2"
	commonctl "github.com/vmware-tanzu/nsx-operator/pkg/controllers/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/util"
)

var (
	log                           = logger.Log
	MarkedForDelete               = true
	EnforceRevisionCheckParam     = false
	ResourceTypeIPPool            = common.ResourceTypeIPPool
	ResourceTypeIPPoolBlockSubnet = common.ResourceTypeIPPoolBlockSubnet
	NewConverter                  = common.NewConverter
)

type IPPoolService struct {
	common.Service
	ipPoolStore            *IPPoolStore
	ipPoolBlockSubnetStore *IPPoolBlockSubnetStore
}

func InitializeIPPool(service common.Service) (*IPPoolService, error) {
	wg := sync.WaitGroup{}
	wgDone := make(chan bool)
	fatalErrors := make(chan error)

	wg.Add(2)

	ipPoolService := &IPPoolService{Service: service}
	ipPoolService.ipPoolStore = &IPPoolStore{ResourceStore: common.ResourceStore{
		Indexer:     cache.NewIndexer(keyFunc, cache.Indexers{common.TagScopeIPPoolCRUID: indexFunc}),
		BindingType: model.IpAddressPoolBindingType(),
	}}
	ipPoolService.ipPoolBlockSubnetStore = &IPPoolBlockSubnetStore{ResourceStore: common.ResourceStore{
		Indexer:     cache.NewIndexer(keyFunc, cache.Indexers{common.TagScopeIPPoolCRUID: indexFunc}),
		BindingType: model.IpAddressPoolBlockSubnetBindingType(),
	}}

	go ipPoolService.InitializeResourceStore(&wg, fatalErrors, ResourceTypeIPPool, ipPoolService.ipPoolStore)
	go ipPoolService.InitializeResourceStore(&wg, fatalErrors, ResourceTypeIPPoolBlockSubnet, ipPoolService.ipPoolBlockSubnetStore)

	go func() {
		wg.Wait()
		close(wgDone)
	}()
	select {
	case <-wgDone:
		break
	case err := <-fatalErrors:
		close(fatalErrors)
		return ipPoolService, err
	}
	return ipPoolService, nil
}

func (service *IPPoolService) CreateOrUpdateIPPool(obj *v1alpha2.IPPool) (bool, bool, error) {
	nsxIPPool, nsxIPSubnets := service.BuildIPPool(obj)
	if nsxIPPool == nil || len(nsxIPSubnets) == 0 {
		err := util.NoEffectiveOption{Desc: "build ip pool and ip pool subnets failed, check its namespace, ippool type and vpc"}
		return false, false, err
	}
	existingIPPool, existingIPSubnets, err := service.indexedIPPoolAndIPPoolSubnets(obj.UID)
	log.V(1).Info("existing ippool and ip subnets", "existingIPPool", existingIPPool, "existingIPSubnets", existingIPSubnets)
	if err != nil {
		log.Error(err, "failed to get ip pool and ip pool subnets by UID", "UID", obj.UID)
		return false, false, err
	}
	ipPoolSubnetsUpdated := false
	ipPoolUpdated := common.CompareResource(IpAddressPoolToComparable(existingIPPool), IpAddressPoolToComparable(nsxIPPool))
	changed, stale := common.CompareResources(IpAddressPoolBlockSubnetsToComparable(existingIPSubnets), IpAddressPoolBlockSubnetsToComparable(nsxIPSubnets))
	changedIPSubnets, staleIPSubnets := ComparableToIpAddressPoolBlockSubnets(changed), ComparableToIpAddressPoolBlockSubnets(stale)
	for i := len(staleIPSubnets) - 1; i >= 0; i-- {
		staleIPSubnets[i].MarkedForDelete = &MarkedForDelete
	}
	finalIPSubnets := append(changedIPSubnets, staleIPSubnets...)
	if len(finalIPSubnets) > 0 {
		ipPoolSubnetsUpdated = true
	}

	if err := service.Operate(nsxIPPool, finalIPSubnets, ipPoolUpdated, ipPoolSubnetsUpdated); err != nil {
		return false, false, err
	}

	realizedSubnets, subnetCidrUpdated, e := service.AcquireRealizedSubnetIP(obj)
	if e != nil {
		return false, false, e
	}
	obj.Status.Subnets = realizedSubnets
	return subnetCidrUpdated, ipPoolSubnetsUpdated, nil
}

func (service *IPPoolService) Operate(nsxIPPool *model.IpAddressPool, nsxIPSubnets []*model.IpAddressPoolBlockSubnet, IPPoolUpdated bool, IPPoolSubnetsUpdated bool) error {
	if !(IPPoolUpdated || IPPoolSubnetsUpdated) {
		return nil
	}
	infraIPPool, err := service.WrapHierarchyIPPool(nsxIPPool, nsxIPSubnets)
	if err != nil {
		return err
	}
	// Get IPPool Type from nsxIPPool
	IPPoolType := common.IPPoolTypePublic
	for _, tag := range nsxIPPool.Tags {
		if *tag.Scope == common.TagScopeIPPoolCRType {
			IPPoolType = *tag.Tag
			break
		}
	}

	if IPPoolType == common.IPPoolTypePrivate {
		ns := service.GetIPPoolNamespace(nsxIPPool)
		orgProjects := commonctl.ServiceMediator.GetOrgProjectVPC(ns)
		if len(orgProjects) == 0 {
			err = util.NoEffectiveOption{Desc: "no effective org and project for ippool"}
		} else {
			err = service.NSXClient.ProjectInfraClient.Patch(orgProjects[0].OrgID, orgProjects[0].ProjectID, *infraIPPool,
				&EnforceRevisionCheckParam)
		}
	} else if IPPoolType == common.IPPoolTypePublic {
		err = service.NSXClient.InfraClient.Patch(*infraIPPool, &EnforceRevisionCheckParam)
	} else {
		err = util.NoEffectiveOption{Desc: "no effective ippool type"}
	}
	if err != nil {
		return err
	}
	if IPPoolUpdated {
		err = service.ipPoolStore.Operate(nsxIPPool)
		if err != nil {
			return err
		}
	}
	if IPPoolSubnetsUpdated {
		err = service.ipPoolBlockSubnetStore.Operate(nsxIPSubnets)
		if err != nil {
			return err
		}
	}
	log.V(1).Info("successfully created or updated ippool and ip subnets", "nsxIPPool", nsxIPPool)
	return nil
}

func (service *IPPoolService) AcquireRealizedSubnetIP(obj *v1alpha2.IPPool) ([]v1alpha2.SubnetResult, bool, error) {
	var realizedSubnets []v1alpha2.SubnetResult
	subnetCidrUpdated := false
	for _, subnetRequest := range obj.Spec.Subnets {
		// check if the subnet is already realized
		realized := false
		realizedSubnet := v1alpha2.SubnetResult{Name: subnetRequest.Name}
		for _, statusSubnet := range obj.Status.Subnets {
			if statusSubnet.Name == subnetRequest.Name && statusSubnet.CIDR != "" {
				realizedSubnet.CIDR = statusSubnet.CIDR
				realized = true
				break
			}
		}
		if !realized {
			cidr, err := service.acquireCidr(obj, &subnetRequest, common.RealizeMaxRetries)
			if err != nil {
				return nil, subnetCidrUpdated, err
			}
			if cidr != "" {
				subnetCidrUpdated = true
			}
			realizedSubnet.CIDR = cidr
		}
		realizedSubnets = append(realizedSubnets, realizedSubnet)
	}
	return realizedSubnets, subnetCidrUpdated, nil
}

func (service *IPPoolService) DeleteIPPool(obj interface{}) error {
	var err error
	var nsxIPPool *model.IpAddressPool
	nsxIPSubnets := make([]*model.IpAddressPoolBlockSubnet, 0)
	switch o := obj.(type) {
	case *v1alpha2.IPPool:
		nsxIPPool, nsxIPSubnets = service.BuildIPPool(o)
		if err != nil {
			log.Error(err, "failed to build ip pool", "IPPool", o)
			return err
		}
	case types.UID:
		nsxIPPool, nsxIPSubnets, err = service.indexedIPPoolAndIPPoolSubnets(o)
		if err != nil {
			log.Error(err, "failed to get ip pool and ip pool subnets by UID", "UID", o)
			return err
		}
	}
	nsxIPPool.MarkedForDelete = &MarkedForDelete
	for i := len(nsxIPSubnets) - 1; i >= 0; i-- {
		nsxIPSubnets[i].MarkedForDelete = &MarkedForDelete
	}
	if err := service.Operate(nsxIPPool, nsxIPSubnets, true, true); err != nil {
		return err
	}
	log.V(1).Info("successfully deleted nsxIPPool", "nsxIPPool", nsxIPPool)
	return nil
}

func (service *IPPoolService) acquireCidr(obj *v1alpha2.IPPool, subnetRequest *v1alpha2.SubnetRequest, retry int) (string, error) {
	m, err := service.NSXClient.RealizedEntitiesClient.List(service.buildIPSubnetIntentPath(obj, subnetRequest), nil)
	if err != nil {
		return "", err
	}
	for _, realizedEntity := range m.Results {
		if *realizedEntity.EntityType == "IpBlockSubnet" {
			for _, attr := range realizedEntity.ExtendedAttributes {
				if *attr.Key == "cidr" {
					cidr := attr.Values[0]
					log.V(1).Info("successfully realized ip subnet ip", "subnetRequest.Name", subnetRequest.Name, "cidr", cidr)
					return cidr, nil
				}
			}
		}
	}
	if retry > 0 {
		log.V(1).Info("failed to acquire subnet cidr, retrying...", "subnet request", subnetRequest, "retry", retry)
		time.Sleep(30 * time.Second)
		return service.acquireCidr(obj, subnetRequest, retry-1)
	} else {
		log.V(1).Info("failed to acquire subnet cidr after multiple retries", "subnet request", subnetRequest)
		return "", nil
	}
}

func (service *IPPoolService) ListIPPoolID() sets.String {
	ipPoolSet := service.ipPoolStore.ListIndexFuncValues(common.TagScopeIPPoolCRUID)
	ipPoolSubnetSet := service.ipPoolBlockSubnetStore.ListIndexFuncValues(common.TagScopeIPPoolCRUID)
	return ipPoolSet.Union(ipPoolSubnetSet)
}

// GetIPPoolNamespace Get IPPool's namespace by tags
func (service *IPPoolService) GetIPPoolNamespace(nsxIPPool *model.IpAddressPool) string {
	for _, tag := range nsxIPPool.Tags {
		if *tag.Scope == common.TagScopeNamespace {
			return *tag.Tag
		}
	}
	return ""
}
