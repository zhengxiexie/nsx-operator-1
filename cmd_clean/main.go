/* Copyright © 2022 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package main

import (
	"os"

	"github.com/vmware-tanzu/nsx-operator/pkg/clean"
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
)

// usage:
// ./bin/clean -mgr-ip='*' -cluster='*' -nsx-passwd='***' -nsx-user='*' -thumbprint="***" -log-level=0

var log = logger.Log

func main() {
	err := clean.Clean()
	if err != nil {
		log.Error(err, "failed to clean nsx resources")
		os.Exit(1)
	}
	os.Exit(0)
}
