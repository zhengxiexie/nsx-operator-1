// Delete DLB resources.
// The size of load balancer service can be, SMALL, MEDIUM, LARGE, XLARGE, or DLB.
// The first four sizes are realized on Edge node as a
// centralized load balancer. DLB is realized on each ESXi hypervisor as a distributed load balancer.
// Previously, this cleanup function was implemented in NCP nsx_policy_cleanup.py.
// Now, it is re-implemented in nsx-operator pkg/clean/.
package clean

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func httpGetOrDelete(method string, url string) (map[string]interface{}, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("admin", "Admin!23Admin")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if method == "DELETE" {
		return nil, nil
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func httpGetDLBServices(url string) ([]string, error) {
	var dlbServicesPath []string

	resp, err := httpGetOrDelete("GET", url)
	if err != nil {
		return dlbServicesPath, err
	}

	for _, item := range resp["results"].([]interface{}) {
		if item.(map[string]interface{})["size"].(string) == "DLB" {
			ncpVersionTagExist, ncpClusterTagExist := false, false
			for _, tagItem := range item.(map[string]interface{})["tags"].([]interface{}) {
				if tagItem.(map[string]interface{})["scope"].(string) == "ncp/version" {
					ncpVersionTagExist = true
				}
				if tagItem.(map[string]interface{})["scope"].(string) == "ncp/cluster" {
					ncpClusterTagExist = true
				}
			}
			if ncpClusterTagExist && ncpVersionTagExist {
				// if path not in dlbServicesPath, add it
				path := item.(map[string]interface{})["path"].(string)
				if !stringInSlice(path, dlbServicesPath) {
					dlbServicesPath = append(dlbServicesPath, path)
				}
			}
		}
	}
	return dlbServicesPath, nil
}

func httpGetVirtualServers(url string, dlbServicesPath []string) ([]string, []string, error) {
	var dlbVirtualServersPath []string
	var dlbPoolsPath []string

	resp, err := httpGetOrDelete("GET", url)
	if err != nil {
		return dlbVirtualServersPath, dlbPoolsPath, err
	}
	for _, item := range resp["results"].([]interface{}) {
		lbServicePath := item.(map[string]interface{})["lb_service_path"].(string)
		if stringInSlice(lbServicePath, dlbServicesPath) {
			ncpVersionTagExist, ncpClusterTagExist := false, false
			for _, tagItem := range item.(map[string]interface{})["tags"].([]interface{}) {
				if tagItem.(map[string]interface{})["scope"].(string) == "ncp/version" {
					ncpVersionTagExist = true
				}
				if tagItem.(map[string]interface{})["scope"].(string) == "ncp/cluster" {
					ncpClusterTagExist = true
				}
			}
			if ncpClusterTagExist && ncpVersionTagExist {
				path := item.(map[string]interface{})["path"].(string)
				if !stringInSlice(path, dlbVirtualServersPath) {
					dlbVirtualServersPath = append(dlbVirtualServersPath, path)
				}
				poolPath := item.(map[string]interface{})["pool_path"].(string)
				if !stringInSlice(poolPath, dlbPoolsPath) {
					dlbPoolsPath = append(dlbPoolsPath, poolPath)
				}
			}
		}
	}
	return dlbVirtualServersPath, dlbPoolsPath, nil
}

func TestCleanDLB(t *testing.T) {
	url := "https://10.176.208.161:443/policy/api/v1/infra/lb-services/"
	dlbServicesPath, _ := httpGetDLBServices(url)
	fmt.Println(dlbServicesPath)
	url = "https://10.176.208.161:443/policy/api/v1/infra/lb-virtual-servers/"
	dlbVirtualServersPath, dlbPoolsPath, _ := httpGetVirtualServers(url, dlbServicesPath)
	fmt.Println(dlbVirtualServersPath)
	fmt.Println(dlbPoolsPath)

	// delete virtual servers of dlb services, then dlb services and dlb pools by sequence
	allPaths := append(dlbVirtualServersPath, dlbServicesPath...)
	allPaths = append(allPaths, dlbPoolsPath...)
	for _, path := range allPaths {
		url = "https://10.176.208.161:443/policy/api/v1" + path
		_, err := httpGetOrDelete("DELETE", url)
		if err != nil {
			fmt.Println(err)
		} else {
			log.Info("delete path: " + path)
		}
	}
}
