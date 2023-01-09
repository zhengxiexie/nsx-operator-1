package VPCnetworkconfig

import (
	"sync"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"k8s.io/client-go/tools/cache"

	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
)

type VPCNetworkConfigService struct {
	common.Service
	VPCNetworkConfigStore *VPCNetworkConfigStore
}

func InitializeVpcNetworkConfig(service common.Service) (*VPCNetworkConfigService, error) {
	wg := sync.WaitGroup{}
	wgDone := make(chan bool)
	fatalErrors := make(chan error)

	wg.Add(2)

	VPCNetworkConfigService := &VPCNetworkConfigService{Service: service}
	VPCNetworkConfigService.VPCNetworkConfigStore = &VPCNetworkConfigStore{ResourceStore: common.ResourceStore{
		Indexer:     cache.NewIndexer(keyFunc, cache.Indexers{common.TagScopeVpcNetworkConfigCRUID: indexFunc}),
		BindingType: model.IpAddressPoolBindingType(),
	}}
	go VPCNetworkConfigService.InitializeProjectResourceStore(&wg, fatalErrors, common.ResourceTypeVPCNetworkConfig, VPCNetworkConfigService.VPCNetworkConfigStore)

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		break
	case err := <-fatalErrors:
		close(fatalErrors)
		return VPCNetworkConfigService, err
	}

	return VPCNetworkConfigService, nil
}
