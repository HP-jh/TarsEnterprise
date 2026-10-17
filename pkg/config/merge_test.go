package config

import "testing"

func TestMergeModules(t *testing.T) {
	mods := []map[string]interface{}{
		{"a": 1, "b": map[string]interface{}{"x": 10}},
		{"b": map[string]interface{}{"y": 20}},
		{"c": 3},
	}
	merged, collisions, err := MergeModules(mods)
	if err != nil {
		t.Fatal(err)
	}
	if len(collisions) != 0 {
		t.Logf("collisions: %v", collisions)
	}
	if merged["a"] != 1 {
		t.Error("a missing")
	}
	if merged["c"] != 3 {
		t.Error("c missing")
	}
	bm, ok := merged["b"].(map[string]interface{})
	if !ok {
		t.Fatal("b not map")
	}
	if bm["x"] != 10 || bm["y"] != 20 {
		t.Error("nested merge failed")
	}
}

func TestMergeModules_Collision(t *testing.T) {
	mods := []map[string]interface{}{
		{"a": 1},
		{"a": 2},
	}
	merged, collisions, err := MergeModules(mods)
	if err != nil {
		t.Fatal(err)
	}
	if len(collisions) == 0 {
		t.Error("expected collision")
	}
	if merged["a"] != 2 {
		t.Error("later value should win")
	}
}

func TestValidateNoOverlap(t *testing.T) {
	a := map[string]interface{}{"x": 1, "y": 2}
	b := map[string]interface{}{"y": 3, "z": 4}
	overlaps := ValidateNoOverlap(a, b)
	if len(overlaps) != 1 || overlaps[0] != "y" {
		t.Errorf("unexpected overlaps: %v", overlaps)
	}
}
