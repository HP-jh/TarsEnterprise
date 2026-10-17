package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StableVersions tracks the last known stable version for each repo.
type StableVersions struct {
	TarsSecureGuard     string    `json:"TarsSecureGuard"`
	TarsEnterprise      string    `json:"TarsEnterprise"`
	TarsWingGuard       string    `json:"TarsWingGuard"`
	TarsOpenClawSandbox string    `json:"TarsOpenClawSandbox"`
	TarsVoiceCorrection string    `json:"TarsVoiceCorrection"`
	TarsUITarsBridge    string    `json:"TarsUITarsBridge"`
	LastSync            time.Time `json:"_last_sync"`
}

// DefaultStablePath returns the default path for stable_versions.json.
func DefaultStablePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".tars", "stable_versions.json")
}

// LoadStableVersions reads the stable versions file.
func LoadStableVersions(path string) (*StableVersions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &StableVersions{}, nil
		}
		return nil, err
	}
	var sv StableVersions
	if err := json.Unmarshal(data, &sv); err != nil {
		return nil, fmt.Errorf("parse stable_versions.json: %w", err)
	}
	return &sv, nil
}

// Save writes the stable versions to disk.
func (sv *StableVersions) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sv, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Get returns the stable version for a repo name.
func (sv *StableVersions) Get(repo string) string {
	switch repo {
	case "TarsSecureGuard":
		return sv.TarsSecureGuard
	case "TarsEnterprise":
		return sv.TarsEnterprise
	case "TarsWingGuard":
		return sv.TarsWingGuard
	case "TarsOpenClawSandbox":
		return sv.TarsOpenClawSandbox
	case "TarsVoiceCorrection":
		return sv.TarsVoiceCorrection
	case "TarsUITarsBridge":
		return sv.TarsUITarsBridge
	}
	return ""
}

// Set updates the stable version for a repo name.
func (sv *StableVersions) Set(repo, version string) {
	switch repo {
	case "TarsSecureGuard":
		sv.TarsSecureGuard = version
	case "TarsEnterprise":
		sv.TarsEnterprise = version
	case "TarsWingGuard":
		sv.TarsWingGuard = version
	case "TarsOpenClawSandbox":
		sv.TarsOpenClawSandbox = version
	case "TarsVoiceCorrection":
		sv.TarsVoiceCorrection = version
	case "TarsUITarsBridge":
		sv.TarsUITarsBridge = version
	}
	sv.LastSync = time.Now()
}
