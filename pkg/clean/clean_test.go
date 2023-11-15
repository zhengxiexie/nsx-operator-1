package clean

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func httpGet(url string) (map[string]interface{}, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("admin", "Admin!23Admin")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Failed to close response body")
		}
	}(resp.Body)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func httpDelete(url string) error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth("admin", "Admin!23Admin")

	_, err = client.Do(req)
	return err
}

func checkTagsExist(tags []interface{}) bool {
	requiredTags := map[string]bool{"ncp/version": false, "ncp/cluster": false}

	for _, tagItem := range tags {
		if scope, ok := tagItem.(map[string]interface{})["scope"].(string); ok {
			if _, exists := requiredTags[scope]; exists {
				requiredTags[scope] = true
			}
		}
	}

	for _, exists := range requiredTags {
		if !exists {
			return false
		}
	}
	return true
}

func httpGetDLBServices(url string) ([]string, error) {
	resp, err := httpGet(url)
	if err != nil {
		return nil, err
	}

	var dlbServicesPath []string
	for _, item := range resp["results"].([]interface{}) {
		if item.(map[string]interface{})["size"].(string) == "DLB" && checkTagsExist(item.(map[string]interface{})["tags"].([]interface{})) {
			dlbServicesPath = append(dlbServicesPath, item.(map[string]interface{})["path"].(string))
		}
	}
	return dlbServicesPath, nil
}

func httpGetVirtualServers(url string, dlbServicesPath []string) ([]string, []string, error) {
	resp, err := httpGet(url)
	if err != nil {
		return nil, nil, err
	}

	dlbServices := make(map[string]bool)
	for _, path := range dlbServicesPath {
		dlbServices[path] = true
	}

	var dlbVirtualServersPath, dlbPoolsPath []string
	for _, item := range resp["results"].([]interface{}) {
		if dlbServices[item.(map[string]interface{})["lb_service_path"].(string)] && checkTagsExist(item.(map[string]interface{})["tags"].([]interface{})) {
			dlbVirtualServersPath = append(dlbVirtualServersPath, item.(map[string]interface{})["path"].(string))
			dlbPoolsPath = append(dlbPoolsPath, item.(map[string]interface{})["pool_path"].(string))
		}
	}
	return dlbVirtualServersPath, dlbPoolsPath, nil
}

func TestCleanDLB(t *testing.T) {
	url := "https://10.176.208.161:443/policy/api/v1/infra/lb-services/"
	dlbServicesPath, _ := httpGetDLBServices(url)

	url = "https://10.176.208.161:443/policy/api/v1/infra/lb-virtual-servers/"
	dlbVirtualServersPath, dlbPoolsPath, _ := httpGetVirtualServers(url, dlbServicesPath)

	allPaths := append(dlbVirtualServersPath, dlbServicesPath...)
	allPaths = append(allPaths, dlbPoolsPath...)
	fmt.Println(allPaths)
	for _, path := range allPaths {
		url = "https://10.176.208.161:443/policy/api/v1" + path
		if err := httpDelete(url); err != nil {
			t.Errorf("Failed to delete path: %s, error: %v", path, err)
		}
	}
}