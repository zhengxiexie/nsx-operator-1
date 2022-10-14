package common

import (
	"github.com/vmware/vsphere-automation-sdk-go/runtime/bindings"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/vmware-tanzu/nsx-operator/pkg/config"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx"
	"github.com/vmware-tanzu/nsx-operator/pkg/util"
)

type Service struct {
	Client    client.Client
	NSXClient *nsx.Client
	NSXConfig *config.NSXOperatorConfig
	Store     map[string]cache.Indexer // Cache for all resources by resource type
}

var (
	Converter *bindings.TypeConverter
	Log       = logf.Log.WithName("service")

	ResourceType               = util.ResourceType
	ResourceTypeSecurityPolicy = util.ResourceTypeSecurityPolicy
	ResourceTypeRule           = util.ResourceTypeRule
	ResourceTypeGroup          = util.ResourceTypeGroup
)

func init() {
	Converter = bindings.NewTypeConverter()
	Converter.SetMode(bindings.REST)
}
