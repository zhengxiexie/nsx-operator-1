package securitypolicy

import (
	"reflect"
	"testing"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
)

func Test_IndexFunc(t *testing.T) {
	mId, mTag, mScope := "11111", "11111", "nsx-op/security_policy_cr_uid"
	m := model.Group{
		Id:   &mId,
		Tags: []model.Tag{{Tag: &mTag, Scope: &mScope}},
	}
	s := model.SecurityPolicy{
		Id:   &mId,
		Tags: []model.Tag{{Tag: &mTag, Scope: &mScope}},
	}
	r := model.Rule{
		Id:   &mId,
		Tags: []model.Tag{{Tag: &mTag, Scope: &mScope}},
	}
	type args struct {
		obj interface{}
	}
	t.Run("1", func(t *testing.T) {
		got, _ := indexFuncSecurityPolicy(s)
		if !reflect.DeepEqual(got, []string{"11111"}) {
			t.Errorf("securityPolicyCRUIDScopeIndexFunc() = %v, want %v", got, model.Tag{Tag: &mTag, Scope: &mScope})
		}
	})
	t.Run("2", func(t *testing.T) {
		got, _ := indexFuncGroup(m)
		if !reflect.DeepEqual(got, []string{"11111"}) {
			t.Errorf("securityPolicyCRUIDScopeIndexFunc() = %v, want %v", got, model.Tag{Tag: &mTag, Scope: &mScope})
		}
	})
	t.Run("3", func(t *testing.T) {
		got, _ := indexFuncRule(r)
		if !reflect.DeepEqual(got, []string{"11111"}) {
			t.Errorf("securityPolicyCRUIDScopeIndexFunc() = %v, want %v", got, model.Tag{Tag: &mTag, Scope: &mScope})
		}
	})
}

func Test_filterTag(t *testing.T) {
	mTag, mScope := "11111", "nsx-op/security_policy_cr_uid"
	mTag2, mScope2 := "11111", "nsx"
	tags := []model.Tag{{Scope: &mScope, Tag: &mTag}}
	tags2 := []model.Tag{{Scope: &mScope2, Tag: &mTag2}}
	var res []string
	var res2 []string
	type args struct {
		v   []model.Tag
		res []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"1", args{v: tags, res: res}, []string{"11111"}},
		{"1", args{v: tags2, res: res2}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterTag(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_KeyFunc(t *testing.T) {
	Id := "11111"
	g := model.Group{Id: &Id}
	s := model.SecurityPolicy{Id: &Id}
	r := model.Rule{Id: &Id}
	type args struct {
		obj interface{}
	}
	t.Run("1", func(t *testing.T) {
		got, _ := keyFuncSecurityPolicy(s)
		if got != "11111" {
			t.Errorf("keyFunc() = %v, want %v", got, "11111")
		}
	})
	t.Run("2", func(t *testing.T) {
		got, _ := keyFuncGroup(g)
		if got != "11111" {
			t.Errorf("keyFunc() = %v, want %v", got, "11111")
		}
	})
	t.Run("3", func(t *testing.T) {
		got, _ := keyFuncRule(r)
		if got != "11111" {
			t.Errorf("keyFunc() = %v, want %v", got, "11111")
		}
	})
}
