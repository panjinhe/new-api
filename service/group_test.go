package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
)

func TestGetUserUsableGroupsAddsAutoWhenDefaultAutoEnabled(t *testing.T) {
	originalDefaultUseAutoGroup := setting.DefaultUseAutoGroup
	originalUserUsableGroups := setting.UserUsableGroups2JSONString()
	defer func() {
		setting.DefaultUseAutoGroup = originalDefaultUseAutoGroup
		if err := setting.UpdateUserUsableGroupsByJSONString(originalUserUsableGroups); err != nil {
			t.Fatalf("failed to restore user usable groups: %v", err)
		}
	}()

	if err := setting.UpdateUserUsableGroupsByJSONString(`{"codex-plus":"codex-plus"}`); err != nil {
		t.Fatalf("failed to set user usable groups: %v", err)
	}
	setting.DefaultUseAutoGroup = true

	groups := GetUserUsableGroups("codex-plus")
	if _, ok := groups["auto"]; !ok {
		t.Fatal("expected auto group to be present when DefaultUseAutoGroup is enabled")
	}
	if desc := groups["auto"]; desc != "自动分组" {
		t.Fatalf("unexpected auto group description: got %q want %q", desc, "自动分组")
	}
}

func TestGetUserUsableGroupsDoesNotAddAutoWhenDefaultAutoDisabled(t *testing.T) {
	originalDefaultUseAutoGroup := setting.DefaultUseAutoGroup
	originalUserUsableGroups := setting.UserUsableGroups2JSONString()
	defer func() {
		setting.DefaultUseAutoGroup = originalDefaultUseAutoGroup
		if err := setting.UpdateUserUsableGroupsByJSONString(originalUserUsableGroups); err != nil {
			t.Fatalf("failed to restore user usable groups: %v", err)
		}
	}()

	if err := setting.UpdateUserUsableGroupsByJSONString(`{"codex-plus":"codex-plus"}`); err != nil {
		t.Fatalf("failed to set user usable groups: %v", err)
	}
	setting.DefaultUseAutoGroup = false

	groups := GetUserUsableGroups("codex-plus")
	if _, ok := groups["auto"]; ok {
		t.Fatal("did not expect auto group to be present when DefaultUseAutoGroup is disabled")
	}
}
