package company

import "testing"

func TestGetSchema_AllTypes(t *testing.T) {
	for _, ct := range AllTypes {
		s, err := GetSchema(ct)
		if err != nil {
			t.Errorf("GetSchema(%s) error: %v", ct, err)
			continue
		}
		if s.Type != ct {
			t.Errorf("type mismatch: got %s want %s", s.Type, ct)
		}
		if len(s.Roles) == 0 {
			t.Errorf("%s: no roles", ct)
		}
		if len(s.Resources) == 0 {
			t.Errorf("%s: no resources", ct)
		}
	}
}

func TestValidatePermission(t *testing.T) {
	schema, _ := GetSchema(Education)
	if !ValidatePermission(schema, "teacher", "course", ActionRead) {
		t.Error("teacher should read course")
	}
	if !ValidatePermission(schema, "principal", "grade", ActionWrite) {
		t.Error("principal should have wildcard")
	}
	if ValidatePermission(schema, "student", "grade", ActionWrite) {
		t.Error("student should not write grade")
	}
}

func TestGetSchema_Unknown(t *testing.T) {
	_, err := GetSchema("unknown")
	if err == nil {
		t.Error("expected error for unknown type")
	}
}
