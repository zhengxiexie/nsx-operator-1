/* Copyright © 2024 Broadcom, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package securitypolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetCluster(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "k8scl-one", getCluster(service))
}
