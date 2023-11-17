package clean

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/vmware-tanzu/nsx-operator/pkg/config"
)

type (
	mapInterface = map[string]interface{}
	mapBool      = map[string]bool
)

func httpGet(cf *config.NSXOperatorConfig, url string) (map[string]interface{}, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	log.Info("Getting ", "url", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(cf.NsxApiUser, cf.NsxApiPassword)

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

	var response mapInterface
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func httpDelete(cf *config.NSXOperatorConfig, url string) error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	log.Info("Deleting ", "url", url)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(cf.NsxApiUser, cf.NsxApiPassword)

	_, err = client.Do(req)
	return err
}

func checkTagsExist(tags []interface{}) bool {
	requiredTags := mapBool{"ncp/version": false, "ncp/cluster": false}

	for _, tagItem := range tags {
		if scope, ok := tagItem.(mapInterface)["scope"].(string); ok {
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

func httpGetDLBServices(cf *config.NSXOperatorConfig) ([]string, error) {
	url := "https://" + cf.NsxApiManagers[0] + ":443/policy/api/v1/infra/lb-services/"
	resp, err := httpGet(cf, url)
	if err != nil {
		return nil, err
	}

	var dlbServicesPath []string
	for _, item := range resp["results"].([]interface{}) {
		if item.(mapInterface)["size"].(string) == "DLB" && checkTagsExist(item.(mapInterface)["tags"].([]interface{})) {
			dlbServicesPath = append(dlbServicesPath, item.(mapInterface)["path"].(string))
		}
	}
	return dlbServicesPath, nil
}

func httpGetVirtualServers(cf *config.NSXOperatorConfig, dlbServicesPath []string) ([]string, []string, error) {
	url := "https://" + cf.NsxApiManagers[0] + ":443/policy/api/v1/infra/lb-virtual-servers/"
	resp, err := httpGet(cf, url)
	if err != nil {
		return nil, nil, err
	}

	dlbServices := make(mapBool)
	for _, path := range dlbServicesPath {
		dlbServices[path] = true
	}

	var dlbVirtualServersPath, dlbPoolsPath []string
	for _, item := range resp["results"].([]interface{}) {
		if dlbServices[item.(mapInterface)["lb_service_path"].(string)] && checkTagsExist(item.(mapInterface)["tags"].([]interface{})) {
			dlbVirtualServersPath = append(dlbVirtualServersPath, item.(mapInterface)["path"].(string))
			dlbPoolsPath = append(dlbPoolsPath, item.(mapInterface)["pool_path"].(string))
		}
	}
	return dlbVirtualServersPath, dlbPoolsPath, nil
}

func CleanDLB(cf *config.NSXOperatorConfig) error {
	log.Info("Deleting DLB resources started")
	dlbServicesPath, err := httpGetDLBServices(cf)
	if err != nil {
		return err
	}
	dlbVirtualServersPath, dlbPoolsPath, err := httpGetVirtualServers(cf, dlbServicesPath)
	if err != nil {
		return err
	}

	allPaths := append(dlbVirtualServersPath, dlbServicesPath...)
	allPaths = append(allPaths, dlbPoolsPath...)
	log.Info("Deleting DLB resources", "paths", allPaths)
	for _, path := range allPaths {
		url := "https://" + cf.NsxApiManagers[0] + ":443/policy/api/v1" + path
		if err = httpDelete(cf, url); err != nil {
			return err
		}
	}
	return nil
}
