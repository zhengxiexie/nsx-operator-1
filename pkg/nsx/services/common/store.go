package common

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/vmware/vsphere-automation-sdk-go/lib/vapi/std/errors"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	util2 "github.com/vmware-tanzu/nsx-operator/pkg/nsx/util"
	"github.com/vmware-tanzu/nsx-operator/pkg/util"
)

const (
	PageSize int64 = 1000
)

// OperateStore is the interface for store, it should be implemented by subclass
// TransResourceToStore is the method to transform the resource of type data.StructValue
// to specific nsx-t side resource and then add it to the store.
// CRUDResource is the method to create, update and delete the resource to the store based
// on its tag MarkedForDelete.
type OperateStore interface {
	TransResourceToStore(obj *data.StructValue) error
	CRUDResource(obj interface{}) error
}

func DecrementPageSize(pageSize *int64) {
	*pageSize -= 100
	if int(*pageSize) <= 0 {
		*pageSize = 10
	}
}

func TransError(err error) error {
	switch err.(type) {
	case errors.ServiceUnavailable:
		vApiError, _ := err.(errors.ServiceUnavailable)
		if vApiError.Data == nil {
			return err
		}
		data, errs := Converter.ConvertToGolang(vApiError.Data, model.ApiErrorBindingType())
		if len(errs) > 0 {
			return err
		}
		apiError := data.(model.ApiError)
		if *apiError.ErrorCode == int64(60576) {
			return util2.PageMaxError{Desc: "page max overflow"}
		}
	default:
		return err
	}
	return err
}

func (service *Service) QueryResource(wg *sync.WaitGroup, fatalErrors chan error, resourceTypeValue string, operateStore OperateStore) {
	defer wg.Done()

	tagScopeClusterKey := strings.Replace(util.TagScopeCluster, "/", "\\/", -1)
	tagScopeClusterValue := strings.Replace(service.NSXClient.NsxConfig.Cluster, ":", "\\:", -1)
	tagParam := fmt.Sprintf("tags.scope:%s AND tags.tag:%s", tagScopeClusterKey, tagScopeClusterValue)
	resourceParam := fmt.Sprintf("%s:%s", ResourceType, resourceTypeValue)
	queryParam := resourceParam + " AND " + tagParam

	var cursor *string = nil
	pageSize := PageSize
	for {
		response, err := service.NSXClient.QueryClient.List(queryParam, cursor, nil, &pageSize, nil, nil)
		err = TransError(err)
		if _, ok := err.(util2.PageMaxError); ok == true {
			DecrementPageSize(&pageSize)
			continue
		}
		if err != nil {
			fatalErrors <- err
		}
		for _, entity := range response.Results {
			err = operateStore.TransResourceToStore(entity)
			if err != nil {
				fatalErrors <- err
			}
		}
		cursor = response.Cursor
		if cursor == nil {
			break
		}
		c, _ := strconv.Atoi(*cursor)
		if int64(c) >= *response.ResultCount {
			break
		}
	}
}
