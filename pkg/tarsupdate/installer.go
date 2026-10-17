package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Installer handles atomic binary replacement and self-restart.
type Installer struct {
	// BackupDir is where old binaries are kept before cleanup.
	BackupDir string
}

// NewInstaller creates an installer.
func NewInstaller() *Installer {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}
	return &Installer{
		BackupDir: filepath.Join(home, ".tars", "backups"),
	}
}

// InstallResult describes what happened during install.
type InstallResult struct {
	OldPath     string
	NewPath     string
	BackupPath  string
	NeedsRestart bool
}

// Install replaces the current binary with the downloaded one.
func (ins *Installer) Install(currentExe, downloadedFile string) (*InstallResult, error) {
	if err := os.MkdirAll(ins.BackupDir, 0755); err != nil {
		return nil, fmt.Errorf("backup dir: %w", err)
	}

	// Absolute paths
	currentAbs, err := filepath.Abs(currentExe)
	if err != nil {
		return nil, err
	}
	downloadAbs, err := filepath.Abs(downloadedFile)
	if err != nil {
		return nil, err
	}

	// Create backup of current binary
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s.%s.bak", filepath.Base(currentAbs), timestamp)
	backupPath := filepath.Join(ins.BackupDir, backupName)

	if err := copyFile(currentAbs, backupPath); err != nil {
		return nil, fmt.Errorf("backup current: %w", err)
	}

	// On Windows, cannot replace a running executable directly.
	if runtime.GOOS == "windows" {
		return ins.installWindows(currentAbs, downloadAbs, backupPath)
	}

	// Unix: atomic rename
	tmpNext := currentAbs + ".new"
	if err := copyFile(downloadAbs, tmpNext); err != nil {
		return nil, fmt.Errorf("stage new: %w", err)
	}
	if err := os.Chmod(tmpNext, 0755); err != nil {
		return nil, fmt.Errorf("chmod new: %w", err)
	}
	if err := os.Rename(tmpNext, currentAbs); err != nil {
		// Rollback
		_ = copyFile(backupPath, currentAbs)
		return nil, fmt.Errorf("atomic replace: %w", err)
	}

	return &InstallResult{
		OldPath:      currentAbs,
		NewPath:      currentAbs,
		BackupPath:   backupPath,
		NeedsRestart: true,
	}, nil
}

func (ins *Installer) installWindows(currentAbs, downloadAbs, backupPath string) (*InstallResult, error) {
	// Windows: move current out of the way, then move new in
	tmpOld := currentAbs + ".old"
	_ = os.Remove(tmpOld) // clean previous if exists

	if err := os.Rename(currentAbs, tmpOld); err != nil {
		return nil, fmt.Errorf("rename current: %w", err)
	}
	if err := copyFile(downloadAbs, currentAbs); err != nil {
		// Rollback
		_ = os.Rename(tmpOld, currentAbs)
		return nil, fmt.Errorf("copy new: %w", err)
	}

	// Schedule cleanup of .old on next boot or via helper
	_ = os.Remove(tmpOld) // try immediate cleanup (may fail if running)

	return &InstallResult{
		OldPath:      currentAbs,
		NewPath:      currentAbs,
		BackupPath:   backupPath,
		NeedsRestart: true,
	}, nil
}

// SelfRestart re-executes the current binary.
func SelfRestart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Start()
}

// Rollback restores the backup binary.
func (ins *Installer) Rollback(currentExe, backupPath string) error {
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}
	return copyFile(backupPath, currentExe)
}

// CleanupOldBackups removes backups older than keepDays.
func (ins *Installer) CleanupOldBackups(keepDays int) error {
	entries, err := os.ReadDir(ins.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(ins.BackupDir, e.Name()))
		}
	}
	return nil
}


