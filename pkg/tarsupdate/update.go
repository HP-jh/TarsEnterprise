package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Updater orchestrates the check-download-verify-replace flow.
type Updater struct {
	BinaryName    string           // e.g. "twg", "wing-guard", "knowledge"
	CurrentVersion string          // e.g. "v0.1.0"
	RepoOwner     string           // e.g. "HP-jh"
	RepoName      string           // e.g. "TarsWingGuard"
	InstallDir    string           // directory where binary lives
	BackupDir     string           // directory to keep old binaries
	StableFile    string           // path to stable_versions.json
	GitHub        *GitHubClient
	Platform      Platform
}

// NewUpdater creates an updater for the given binary.
func NewUpdater(binaryName, currentVersion, owner, repo string) *Updater {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".tars")
	return &Updater{
		BinaryName:     binaryName,
		CurrentVersion: currentVersion,
		RepoOwner:      owner,
		RepoName:       repo,
		InstallDir:     filepath.Dir(os.Args[0]),
		BackupDir:      filepath.Join(baseDir, "backups"),
		StableFile:     filepath.Join(baseDir, "stable_versions.json"),
		GitHub:         NewGitHubClient(owner, repo),
		Platform:       CurrentPlatform(),
	}
}

// CheckResult holds the outcome of a version check.
type CheckResult struct {
	HasUpdate   bool
	Current     Version
	Latest      Version
	Release     ReleaseInfo
	AssetName   string
	AssetURL    string
	Error       error
}

// Check queries GitHub for a newer release.
func (u *Updater) Check() CheckResult {
	cur, err := ParseVersion(u.CurrentVersion)
	if err != nil {
		return CheckResult{Error: fmt.Errorf("parse current version: %w", err)}
	}

	rel, err := u.GitHub.LatestRelease()
	if err != nil {
		return CheckResult{Error: fmt.Errorf("fetch latest release: %w", err)}
	}

	latest, err := ParseVersion(rel.TagName)
	if err != nil {
		return CheckResult{Error: fmt.Errorf("parse latest version: %w", err)}
	}

	assetName := u.Platform.AssetName(u.BinaryName, rel.TagName)
	assetURL, _ := rel.FindAssetURL(assetName)

	return CheckResult{
		HasUpdate: latest.IsNewerThan(cur),
		Current:   cur,
		Latest:    latest,
		Release:   rel,
		AssetName: assetName,
		AssetURL:  assetURL,
	}
}

// Apply downloads, verifies, and replaces the binary.
func (u *Updater) Apply(cr CheckResult) error {
	if !cr.HasUpdate {
		return fmt.Errorf("no update available")
	}
	if cr.AssetURL == "" {
		return fmt.Errorf("no asset URL for %s", cr.AssetName)
	}

	// Ensure dirs exist
	if err := os.MkdirAll(u.BackupDir, 0755); err != nil {
		return fmt.Errorf("mkdir backups: %w", err)
	}

	tmpDir := filepath.Join(u.BackupDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("mkdir tmp: %w", err)
	}

	// Download new binary
	newBinaryPath := filepath.Join(tmpDir, cr.AssetName)
	if err := u.GitHub.DownloadAsset(cr.AssetURL, newBinaryPath); err != nil {
		return fmt.Errorf("download asset: %w", err)
	}

	// Download and verify SHA256
	sums, err := u.GitHub.DownloadSHA256SUMS(cr.Release)
	if err != nil {
		return fmt.Errorf("download SHA256SUMS: %w", err)
	}
	expectedHash, ok := sums[cr.AssetName]
	if !ok {
		return fmt.Errorf("no hash found for %s", cr.AssetName)
	}
	actualHash, err := fileSHA256(newBinaryPath)
	if err != nil {
		return fmt.Errorf("hash new binary: %w", err)
	}
	if !strings.EqualFold(actualHash, expectedHash) {
		_ = os.Remove(newBinaryPath)
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	// Make executable
	if err := os.Chmod(newBinaryPath, 0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	// Backup current binary
	currentBinary := filepath.Join(u.InstallDir, u.BinaryName)
	if runtime.GOOS == "windows" {
		currentBinary += ".exe"
	}
	backupPath := filepath.Join(u.BackupDir, fmt.Sprintf("%s-%s-%s", u.BinaryName, u.CurrentVersion, time.Now().Format("20060102T150405")))
	if runtime.GOOS == "windows" {
		backupPath += ".exe"
	}
	if fileExists(currentBinary) {
		if err := copyFile(currentBinary, backupPath); err != nil {
			return fmt.Errorf("backup current binary: %w", err)
		}
	}

	// Atomic replace
	tmpTarget := currentBinary + ".tmp"
	if err := copyFile(newBinaryPath, tmpTarget); err != nil {
		return fmt.Errorf("stage new binary: %w", err)
	}
	if err := renameFile(tmpTarget, currentBinary); err != nil {
		_ = os.Remove(tmpTarget)
		return fmt.Errorf("replace binary: %w", err)
	}

	// Clean up temp download
	_ = os.Remove(newBinaryPath)

	// Update stable_versions.json
	stable := loadStableVersions(u.StableFile)
	stable[u.RepoName] = cr.Latest.String()
	stable["_last_sync"] = time.Now().Format(time.RFC3339)
	_ = saveStableVersions(u.StableFile, stable)

	return nil
}

// Rollback restores the most recent backup.
func (u *Updater) Rollback() error {
	entries, err := os.ReadDir(u.BackupDir)
	if err != nil {
		return fmt.Errorf("read backups: %w", err)
	}

	var latestBackup string
	var latestTime time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasPrefix(e.Name(), u.BinaryName+"-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestBackup = filepath.Join(u.BackupDir, e.Name())
		}
	}

	if latestBackup == "" {
		return fmt.Errorf("no backup found")
	}

	currentBinary := filepath.Join(u.InstallDir, u.BinaryName)
	if runtime.GOOS == "windows" {
		currentBinary += ".exe"
	}

	tmpTarget := currentBinary + ".tmp"
	if err := copyFile(latestBackup, tmpTarget); err != nil {
		return fmt.Errorf("stage rollback: %w", err)
	}
	if err := renameFile(tmpTarget, currentBinary); err != nil {
		_ = os.Remove(tmpTarget)
		return fmt.Errorf("replace with backup: %w", err)
	}
	return nil
}

// FallbackToStable returns the locally-cached stable version when GitHub is unreachable.
func (u *Updater) FallbackToStable() (string, error) {
	stable := loadStableVersions(u.StableFile)
	v, ok := stable[u.RepoName]
	if !ok {
		return "", fmt.Errorf("no stable version cached for %s", u.RepoName)
	}
	return v, nil
}

// file helpers

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func renameFile(old, new string) error {
	if runtime.GOOS == "windows" {
		// Windows does not allow overwriting via rename; remove first
		_ = os.Remove(new)
	}
	return os.Rename(old, new)
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func loadStableVersions(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]string)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return make(map[string]string)
	}
	return m
}

func saveStableVersions(path string, m map[string]string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
