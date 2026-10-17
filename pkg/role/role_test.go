package role

import (
	"testing"

	"github.com/HP-jh/TarsEnterprise/pkg/company"
)

func TestManager_RegisterAndCheck(t *testing.T) {
	m := NewManager()
	_ = m.BuildMatrix(company.Tech)

	id := &Identity{UserID: "u1", RoleID: "dev", CompanyType: company.Tech}
	if err := m.RegisterIdentity(id); err != nil {
		t.Fatal(err)
	}

	if !m.Can("u1", "code", company.ActionWrite) {
		t.Error("dev should write code")
	}
	if m.Can("u1", "people", company.ActionWrite) {
		t.Error("dev should not write people")
	}
	if m.Can("unknown", "code", company.ActionRead) {
		t.Error("unknown user should not have permission")
	}
}

func TestManager_BuildMatrix(t *testing.T) {
	m := NewManager()
	for _, ct := range company.AllTypes {
		if err := m.BuildMatrix(ct); err != nil {
			t.Errorf("BuildMatrix(%s): %v", ct, err)
		}
	}
}

func TestBindToProfession(t *testing.T) {
	cases := map[string]string{
		"dev":      "sapiens-engineer",
		"teacher":  "sapiens-educator",
		"ceo":      "sapiens-executive",
	}
	for role, wantSuffix := range cases {
		got := BindToProfession(role)
		if len(got) < len(wantSuffix) || got[len(got)-len(wantSuffix):] != wantSuffix {
			t.Errorf("BindToProfession(%s) = %s, want suffix %s", role, got, wantSuffix)
		}
	}
}
