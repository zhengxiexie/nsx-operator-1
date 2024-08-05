package subnetport

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/crd.nsx.vmware.com/v1alpha1"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/util"
)

var (
	String = common.String
)

func (service *SubnetPortService) buildSubnetPort(obj interface{}, nsxSubnet *model.VpcSubnet, contextID string, labelTags *map[string]string, isVmSubnetPort bool) (*model.VpcSubnetPort, error) {
	var objName, objNamespace, uid, appId, allocateAddresses string
	switch o := obj.(type) {
	case *v1alpha1.SubnetPort:
		objName = o.Name
		objNamespace = o.Namespace
		uid = string(o.UID)
	case *corev1.Pod:
		objName = o.Name
		objNamespace = o.Namespace
		uid = string(o.UID)
		appId = string(o.UID)
	}
	if *nsxSubnet.DhcpConfig.EnableDhcp {
		allocateAddresses = "DHCP"
	} else {
		allocateAddresses = "BOTH"
	}
	nsxSubnetPortName := util.GenerateDisplayName(objName, "port", "", "", "")
	nsxSubnetPortID := util.GenerateID(uid, "", "", "")
	// use the subnetPort CR UID as the attachment uid generation to ensure the latter stable
	nsxCIFID, err := uuid.NewRandomFromReader(bytes.NewReader([]byte(nsxSubnetPortID)))
	if err != nil {
		return nil, err
	}
	nsxSubnetPortPath := fmt.Sprintf("%s/ports/%s", *nsxSubnet.Path, uid)
	if err != nil {
		return nil, err
	}
	namespace := &corev1.Namespace{}
	namespacedName := types.NamespacedName{
		Name: objNamespace,
	}
	if err := service.Client.Get(context.Background(), namespacedName, namespace); err != nil {
		return nil, err
	}
	namespace_uid := namespace.UID
	tags := util.BuildBasicTags(getCluster(service), obj, namespace_uid)
	var tagsFiltered []model.Tag
	for _, tag := range tags {
		if isVmSubnetPort && *tag.Scope == common.TagScopeNamespaceUID {
			continue
		}
		if !isVmSubnetPort && *tag.Scope == common.TagScopeVMNamespaceUID {
			continue
		}
		tagsFiltered = append(tagsFiltered, tag)
	}

	if labelTags != nil {
		for k, v := range *labelTags {
			tagsFiltered = append(tagsFiltered, model.Tag{Scope: String(k), Tag: String(v)})
		}
	}
	nsxSubnetPort := &model.VpcSubnetPort{
		DisplayName: String(nsxSubnetPortName),
		Id:          String(nsxSubnetPortID),
		Attachment: &model.PortAttachment{
			AllocateAddresses: &allocateAddresses,
			Id:                String(nsxCIFID.String()),
			TrafficTag:        common.Int64(0),
			Type_:             String("STATIC"),
		},
		Tags:       tagsFiltered,
		Path:       &nsxSubnetPortPath,
		ParentPath: nsxSubnet.Path,
	}
	if appId != "" {
		nsxSubnetPort.Attachment.AppId = &appId
	}
	nsxSubnetPort.Attachment.ContextId = &contextID
	return nsxSubnetPort, nil
}

func getCluster(service *SubnetPortService) string {
	return service.NSXConfig.Cluster
}
