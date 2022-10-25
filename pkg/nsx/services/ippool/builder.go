package ippool

import (
	"strings"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha2"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/util"
)

var (
	Int64  = common.Int64
	String = common.String
)

func (service *IPPoolService) BuildIPPool(IPPool *v1alpha2.IPPool) (*model.IpAddressPool, []*model.IpAddressPoolBlockSubnet) {
	return &model.IpAddressPool{
		Id:          String(service.buildIPPoolID(IPPool)),
		DisplayName: String(service.buildIPPoolName(IPPool)),
		Tags:        service.buildIPPoolTags(IPPool),
	}, service.buildIPSubnets(IPPool)
}

func (service *IPPoolService) buildIPPoolID(IPPool *v1alpha2.IPPool) string {
	return strings.Join([]string{"ipc", string(IPPool.UID)}, "_")
}

func (service *IPPoolService) buildIPPoolName(IPPool *v1alpha2.IPPool) string {
	return strings.Join([]string{"ipc", getCluster(service), string(IPPool.UID), IPPool.ObjectMeta.Name}, "-")
}

func (service *IPPoolService) buildIPPoolTags(IPPool *v1alpha2.IPPool) []model.Tag {
	return []model.Tag{
		{Scope: String(common.TagScopeCluster), Tag: String(getCluster(service))},
		{Scope: String(common.TagScopeNamespace), Tag: String(IPPool.ObjectMeta.Namespace)},
		{Scope: String(common.TagScopeIPPoolCRName), Tag: String(IPPool.ObjectMeta.Name)},
		{Scope: String(common.TagScopeIPPoolCRUID), Tag: String(string(IPPool.UID))},
	}
}

func (service *IPPoolService) buildIPSubnets(IPPool *v1alpha2.IPPool) []*model.IpAddressPoolBlockSubnet {
	var IPSubnets []*model.IpAddressPoolBlockSubnet
	for _, subnetRequest := range IPPool.Spec.Subnets {
		IPSubnet := service.buildIPSubnet(IPPool, subnetRequest)
		IPSubnets = append(IPSubnets, IPSubnet)
	}
	return IPSubnets
}

func (service *IPPoolService) buildIPSubnetID(IPPool *v1alpha2.IPPool, subnetRequest *v1alpha2.SubnetRequest) string {
	return strings.Join([]string{"ibs", string(IPPool.UID), subnetRequest.Name}, "_")
}

func (service *IPPoolService) buildIPSubnetName(IPPool *v1alpha2.IPPool, subnetRequest *v1alpha2.SubnetRequest) string {
	return strings.Join([]string{"ibs", IPPool.Name, subnetRequest.Name}, "-")
}

func (service *IPPoolService) buildIPSubnetTags(IPPool *v1alpha2.IPPool, subnetRequest *v1alpha2.SubnetRequest) []model.Tag {
	return []model.Tag{
		{Scope: String(common.TagScopeCluster), Tag: String(getCluster(service))},
		{Scope: String(common.TagScopeNamespace), Tag: String(IPPool.ObjectMeta.Namespace)},
		{Scope: String(common.TagScopeIPPoolCRName), Tag: String(IPPool.ObjectMeta.Name)},
		{Scope: String(common.TagScopeIPPoolCRUID), Tag: String(string(IPPool.UID))},
		{Scope: String(common.TagScopeIPSubnetName), Tag: String(subnetRequest.Name)},
	}
}

func (service *IPPoolService) buildIPSubnetIntentPath(IPPool *v1alpha2.IPPool, subnetRequest *v1alpha2.SubnetRequest) string {
	return strings.Join([]string{"/orgs/default/projects/project-1/infra/ip-pools", service.buildIPPoolID(IPPool),
		"ip-subnets", service.buildIPSubnetID(IPPool, subnetRequest)}, "/")
}

func (service *IPPoolService) buildIPSubnet(IPPool *v1alpha2.IPPool, subnetRequest v1alpha2.SubnetRequest) *model.IpAddressPoolBlockSubnet {
	return &model.IpAddressPoolBlockSubnet{
		Id:          String(service.buildIPSubnetID(IPPool, &subnetRequest)),
		DisplayName: String(service.buildIPSubnetName(IPPool, &subnetRequest)),
		Tags:        service.buildIPSubnetTags(IPPool, &subnetRequest),
		Size:        Int64(util.CalculateSubnetSize(subnetRequest.PrefixLength)),
		IpBlockPath: String("/orgs/default/projects/project-1/infra/ip-blocks/block-test"),
	}
}
