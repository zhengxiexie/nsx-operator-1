package e2e

import (
	"log"
	"path/filepath"
	"strings"
	"testing"
)

const (
	NetworkInfoCRType     = "networkinfoes"
	NSCRType              = "namespaces"
	PrivateIPBlockNSXType = "IpAddressBlock"

	InfraVPCNamespace       = "kube-system"
	SharedInfraVPCNamespace = "kube-public"

	DefaultPrivateCIDR1    = "172.28.0.0"
	DefaultPrivateCIDR2    = "172.38.0.0"
	InfraPrivateCIDR1      = "172.27.0.0"
	InfraPrivateCIDR2      = "172.37.0.0"
	CustomizedPrivateCIDR1 = "172.29.0.0"
	CustomizedPrivateCIDR2 = "172.39.0.0"
)

var (
	verify_keys = []string{"defaultSNATIP", "lbSubnetCIDR", "lbSubnetPath", "nsxResourcePath"}
)

func verifyCRCreated(t *testing.T, crtype string, ns string, expect int) (string, string) {
	// there should be one networkinfo created
	resources, err := testData.getCRResource(defaultTimeout, crtype, ns)
	// only one networkinfo should be created under ns using default network config
	if len(resources) != expect {
		log.Printf("NetworkInfo list %s size not the same as expected %d", resources, expect)
		panic("NetworkInfo CR creation verify failed")
	}
	assertNil(t, err)

	cr_name, cr_uid := "", ""
	// waiting for CR to be ready
	for k, v := range resources {
		cr_name = k
		cr_uid = strings.TrimSpace(v)
	}

	return cr_name, cr_uid
}

func verifyPrivateIPBlockCreated(t *testing.T, ns, id string) {
	err := testData.waitForResourceExistById(ns, PrivateIPBlockNSXType, id, true)
	assertNil(t, err)
}

func verifyPrivateIPBlockDeleted(t *testing.T, ns, id string) {
	err := testData.waitForResourceExistById(ns, PrivateIPBlockNSXType, id, false)
	assertNil(t, err)
}

// Test Customized NetworkInfo
func TestCustomizedNetworkInfo(t *testing.T) {
	// Create customized networkconfig
	ncPath, _ := filepath.Abs("./manifest/testVPC/customize_networkconfig.yaml")
	_ = applyYAML(ncPath, "")
	nsPath, _ := filepath.Abs("./manifest/testVPC/customize_ns.yaml")
	_ = applyYAML(nsPath, "")

	defer deleteYAML(nsPath, "")
	defer deleteYAML(ncPath, "")

	ns := "customized-ns"

	networkinfo_name, _ := verifyCRCreated(t, NetworkInfoCRType, ns, 1)
	_, ns_uid := verifyCRCreated(t, NSCRType, InfraVPCNamespace, 1)

	vpcPath, err := testData.getCRPropertiesByJson(defaultTimeout, NetworkInfoCRType, networkinfo_name, ns, ".vpcs[0].vpcPath")
	err = testData.waitForResourceExistByPath(vpcPath, true)
	assertNil(t, err)

	//verify private ipblocks created for vpc
	p_ipb_id1 := ns_uid + "_" + CustomizedPrivateCIDR1
	p_ipb_id2 := ns_uid + "_" + CustomizedPrivateCIDR2

	verifyPrivateIPBlockCreated(t, ns, p_ipb_id1)
	verifyPrivateIPBlockCreated(t, ns, p_ipb_id2)
}

// Test Infra NetworkInfo
func TestInfraNetworkInfo(t *testing.T) {
	// Check namespace cr existence
	_, ns_uid := verifyCRCreated(t, NSCRType, InfraVPCNamespace, 1)
	// Check networkinfo cr existence
	networkinfo_name, _ := verifyCRCreated(t, NetworkInfoCRType, InfraVPCNamespace, 1)

	vpcPath, err := testData.getCRPropertiesByJson(defaultTimeout, NetworkInfoCRType, networkinfo_name, InfraVPCNamespace, ".vpcs[0].vpcPath")

	err = testData.waitForResourceExistByPath(vpcPath, true)
	assertNil(t, err)

	//verify private ipblocks created for vpc
	p_ipb_id1 := ns_uid + "_" + InfraPrivateCIDR1
	p_ipb_id2 := ns_uid + "_" + InfraPrivateCIDR2

	verifyPrivateIPBlockCreated(t, InfraVPCNamespace, p_ipb_id1)
	verifyPrivateIPBlockCreated(t, InfraVPCNamespace, p_ipb_id2)

	// kube-public vpcpath should be the same as kube-system vpcpath
	vpcPath2, err := testData.getCRPropertiesByJson(defaultTimeout, NetworkInfoCRType, networkinfo_name, SharedInfraVPCNamespace,
		".vpcs[0].vpcPath")
	assertNil(t, err)
	assertTrue(t, vpcPath == vpcPath2, "vpcPath %s should be the same as vpcPath2 %s", vpcPath, vpcPath2)
}

// Test Default NetworkInfo
func TestDefaultNetworkInfo(t *testing.T) {
	// If no annotation on namespace, then NetworkInfo will use default network config to create vpc under each ns
	ns := "networkinfo-default-1"
	setupTest(t, ns)
	defer teardownTest(t, ns, defaultTimeout)

	// Check namespace cr existence
	_, ns_uid := verifyCRCreated(t, NSCRType, ns, 1)
	// Check networkinfo cr existence
	networkinfo_name, _ := verifyCRCreated(t, NetworkInfoCRType, ns, 1)

	vpcPath, err := testData.getCRPropertiesByJson(defaultTimeout, NetworkInfoCRType, networkinfo_name, ns, ".vpcs[0].vpcPath")

	err = testData.waitForResourceExistByPath(vpcPath, true)
	assertNil(t, err)

	//verify private ipblocks created for vpc, id is nsuid + cidr
	p_ipb_id1 := ns_uid + "_" + DefaultPrivateCIDR1
	p_ipb_id2 := ns_uid + "_" + DefaultPrivateCIDR2

	verifyPrivateIPBlockCreated(t, ns, p_ipb_id1)
	verifyPrivateIPBlockCreated(t, ns, p_ipb_id2)

	// delete namespace and check all resources are deleted
	err = testData.deleteNamespace(ns, defaultTimeout)
	assertNil(t, err)
	// Check namespace cr not existed
	_, _ = verifyCRCreated(t, NSCRType, ns, 0)
	// Check networkinfo cr not existed
	_, _ = verifyCRCreated(t, NetworkInfoCRType, ns, 0)
	err = testData.waitForResourceExistByPath(vpcPath, false)
	assertNil(t, err)
	verifyPrivateIPBlockDeleted(t, ns, p_ipb_id1)
	verifyPrivateIPBlockDeleted(t, ns, p_ipb_id2)

}

// TODO add new ns which share infrastructure, and delete it, vpc should not be deleted

// TODO update vpcnetworkconfig, and check vpc is updated
