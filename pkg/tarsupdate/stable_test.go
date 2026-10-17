package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveStableVersions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stable_versions.json")

	// Load non-existent -> empty
	sv, err := LoadStableVersions(path)
	if err != nil {
		t.Fatal(err)
	}
	if sv.Get("TarsVoiceCorrection") != "" {
		t.Fatal("expected empty")
	}

	// Set and save
	sv.Set("TarsVoiceCorrection", "v0.1.0")
	sv.Set("TarsSecureGuard", "v4.5.3")
	if err := sv.Save(path); err != nil {
		t.Fatal(err)
	}

	// Reload
	sv2, err := LoadStableVersions(path)
	if err != nil {
		t.Fatal(err)
	}
	if sv2.Get("TarsVoiceCorrection") != "v0.1.0" {
		t.Fatalf("got %q", sv2.Get("TarsVoiceCorrection"))
	}
	if sv2.Get("TarsSecureGuard") != "v4.5.3" {
		t.Fatalf("got %q", sv2.Get("TarsSecureGuard"))
	}
	if sv2.LastSync.IsZero() {
		t.Fatal("expected LastSync set")
	}
}

func TestStableVersionsGetSetUnknown(t *testing.T) {
	sv := &StableVersions{}
	sv.Set("UnknownRepo", "v9.9.9")
	if sv.Get("UnknownRepo") != "" {
		t.Fatal("unknown repo should return empty")
	}
}

func TestDefaultStablePath(t *testing.T) {
	p := DefaultStablePath()
	if p == "" {
		t.Fatal("expected non-empty path")
	}
	if filepath.Base(p) != "stable_versions.json" {
		t.Fatalf("unexpected basename: %s", filepath.Base(p))
	}
}

func TestStableVersionsCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stable_versions.json")
	os.WriteFile(path, []byte("not json"), 0644)
	_, err := LoadStableVersions(path)
	if err == nil {
		t.Fatal("expected error for corrupted file")
	}
}
