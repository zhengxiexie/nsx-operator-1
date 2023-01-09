package VPCnetworkconfig

import (
	"errors"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
)

func keyFunc(obj interface{}) (string, error) {
	switch v := obj.(type) {
	case model.Vpc:
		return *v.Id, nil
	default:
		return "", errors.New("keyFunc doesn't support unknown type")
	}
}

func indexFunc(obj interface{}) ([]string, error) {
	res := make([]string, 0, 5)
	switch v := obj.(type) {
	case model.Vpc:
		return filterTag(v.Tags), nil
	default:
		return res, errors.New("indexFunc doesn't support unknown type")
	}
}

var filterTag = func(v []model.Tag) []string {
	res := make([]string, 0, 5)
	for _, tag := range v {
		if *tag.Scope == common.TagScopeVpcNetworkConfigCRUID {
			res = append(res, *tag.Tag)
		}
	}
	return res
}

type VPCNetworkConfigStore struct {
	common.ResourceStore
}
