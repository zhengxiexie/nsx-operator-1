/* Copyright © 2023 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package clean

import (
	"fmt"
	"net/http"

	"k8s.io/client-go/util/retry"

	"github.com/vmware-tanzu/nsx-operator/pkg/config"
	commonctl "github.com/vmware-tanzu/nsx-operator/pkg/controllers/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/ippool"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/securitypolicy"
	sr "github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/staticroute"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/subnet"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/subnetport"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/vpc"
)

var log = logger.Log

// Clean cleans up NSX resources,
// including security policy, static route, subnet, subnet port, subnet set, vpc, ip pool, nsx service account
// it is usually used when nsx-operator is uninstalled and remove all the resources created by nsx-operator
// return error if any, return nil if no error
func Clean(cf *config.NSXOperatorConfig, client *http.Client) (Status, error) {
	log.Info("starting NSX cleanup")
	if err := cf.ValidateConfigFromCmd(); err != nil {
		return ValidationFailed, err
	}
	nsxClient := nsx.GetClient(cf, client)
	if nsxClient == nil {
		return GetNSXClientFailed, fmt.Errorf("failed to get nsx client")
	}
	if cleanupService, err := InitializeCleanupService(cf, nsxClient); err != nil {
		return InitCleanupServiceFailed, err

	} else if cleanupService.err != nil {
		return InitCleanupServiceFailed, cleanupService.err
	} else {
		for _, clean := range cleanupService.cleans {
			if err := retry.OnError(retry.DefaultRetry, func(err error) bool {
				if err != nil {
					log.Info("retrying to clean up NSX resources", "error", err)
					return true
				}
				return false
			}, func() error {
				if err := clean.Cleanup(); err != nil {
					return err
				}
				return nil
			}); err != nil {
				return CleanupResourceFailed, err
			}
		}
	}
	log.Info("cleanup NSX resources successfully")
	return OK, nil
}

// InitializeCleanupService initializes all the CR services
func InitializeCleanupService(cf *config.NSXOperatorConfig, nsxClient *nsx.Client) (*CleanupService, error) {
	cleanupService := NewCleanupService()

	var commonService = common.Service{
		NSXClient: nsxClient,
		NSXConfig: cf,
	}
	vpcService, vpcErr := vpc.InitializeVPC(commonService)
	commonctl.ServiceMediator.VPCService = vpcService

	// initialize all the CR services
	// Use Fluent Interface to escape error check hell

	wrapInitializeSubnetService := func(service common.Service) cleanupFunc {
		return func() (cleanup, error) {
			return subnet.InitializeSubnetService(service)
		}
	}
	wrapInitializeSecurityPolicy := func(service common.Service) cleanupFunc {
		return func() (cleanup, error) {
			return securitypolicy.InitializeSecurityPolicy(service)
		}
	}
	wrapInitializeIPPool := func(service common.Service) cleanupFunc {
		return func() (cleanup, error) {
			return ippool.InitializeIPPool(service)
		}
	}

	wrapInitializeVPC := func(service common.Service) cleanupFunc {
		return func() (cleanup, error) {
			return vpcService, vpcErr
		}
	}

	wrapInitializeStaticRoute := func(service common.Service) cleanupFunc {
		return func() (cleanup, error) {
			return sr.InitializeStaticRoute(service)
		}
	}

	wrapInitializeSubnetPort := func(service common.Service) cleanupFunc {
		return func() (cleanup, error) {
			return subnetport.InitializeSubnetPort(service)
		}
	}
	// TODO: initialize other CR services
	cleanupService = cleanupService.
		AddCleanupService(wrapInitializeSubnetPort(commonService)).
		AddCleanupService(wrapInitializeSubnetService(commonService)).
		AddCleanupService(wrapInitializeSecurityPolicy(commonService)).
		AddCleanupService(wrapInitializeIPPool(commonService)).
		AddCleanupService(wrapInitializeStaticRoute(commonService)).
		AddCleanupService(wrapInitializeVPC(commonService))

	return cleanupService, nil
}
