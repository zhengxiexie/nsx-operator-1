package common

import (
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data/serializers/cleanjson"
)

var log = Log.WithName("compare")

type Comparable interface {
	Simplify() Comparable
	Key() string
	GetDataValue__() (data.DataValue, []error)
}

func CompareResource(existing Comparable, expected Comparable) (isChanged bool, actual Comparable) {
	r1, _ := existing.Simplify().GetDataValue__()
	r2, _ := expected.Simplify().GetDataValue__()
	var dataValueToJSONEncoder = cleanjson.NewDataValueToJsonEncoder()
	s1, _ := dataValueToJSONEncoder.Encode(r1)
	s2, _ := dataValueToJSONEncoder.Encode(r2)
	if s1 != s2 {
		return true, expected
	}
	return false, expected
}

func CompareResources(existing []Comparable, expected []Comparable) (changed []Comparable, stale []Comparable) {
	stale = make([]Comparable, 0)
	changed = make([]Comparable, 0)

	expectedMap := make(map[string]Comparable)
	for _, e := range expected {
		expectedMap[e.Key()] = e
	}
	existingMap := make(map[string]Comparable)
	for _, e := range existing {
		existingMap[e.Key()] = e
	}

	for key, e := range expectedMap {
		if e2, ok := existingMap[key]; ok {
			if isChanged, _ := CompareResource(e2, e); !isChanged {
				continue
			}
		}
		changed = append(changed, e)
	}
	for key, e := range existingMap {
		if _, ok := expectedMap[key]; !ok {
			stale = append(stale, e)
		}
	}
	log.V(1).Info("resources differ", "stale", stale, "changed", changed)
	return changed, stale
}
