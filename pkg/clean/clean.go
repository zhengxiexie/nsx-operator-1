/* Copyright © 2022 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package clean

import (
	"flag"
	"fmt"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/vmware-tanzu/nsx-operator/pkg/config"
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/securitypolicy"
)

var (
	log          = logger.Log
	cf           *config.NSXOperatorConfig
	mgrIp        string
	nsxUser      string
	nsxPasswd    string
	thumbprint   string
	vmcaCertFile string
	cluster      string
)

func init() {
	flag.StringVar(&mgrIp, "mgr-ip", "", "nsx manager ip")
	flag.StringVar(&nsxUser, "nsx-user", "", "nsx username")
	flag.StringVar(&nsxPasswd, "nsx-passwd", "", "nsx password")
	flag.StringVar(&thumbprint, "thumbprint", "", "nsx thumbprint")
	flag.StringVar(&vmcaCertFile, "vmca-cert-file", "", "vmca cert file")
	flag.StringVar(&cluster, "cluster", "", "cluster name")
	flag.IntVar(&config.LogLevel, "log-level", 0, "Use zap-core log system.")
	flag.Parse()

	logf.SetLogger(logger.ZapLogger())
	cf = config.NewNSXOpertorConfig()
	cf.NsxApiManagers = []string{mgrIp}
	cf.NsxApiUser = nsxUser
	cf.NsxApiPassword = nsxPasswd
	cf.Thumbprint = []string{thumbprint}
	cf.NsxApiCertFile = vmcaCertFile
	cf.Cluster = cluster
}

func Clean() error {
	log.Info("starting NSX cleanup")
	var err error
	var securityPolicyService *securitypolicy.SecurityPolicyService

	nsxClient := nsx.GetClient(cf)
	if nsxClient == nil {
		return fmt.Errorf("failed to get nsx client")
	}

	var commonService = common.Service{
		NSXClient: nsxClient,
		NSXConfig: cf,
	}

	if securityPolicyService, err = securitypolicy.InitializeSecurityPolicy(commonService); err != nil {
		return fmt.Errorf("failed to initialize securitypolicy service: %v", err)
	}
	err = securityPolicyService.Cleanup()
	if err != nil {
		return fmt.Errorf("failed to clean up: %v", err)
	}

	log.Info("successfully to cleanup NSX resources")
	return nil
}
