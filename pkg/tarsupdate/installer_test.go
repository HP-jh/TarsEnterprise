package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	data := []byte("hello installer")
	os.WriteFile(src, data, 0644)

	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("copy mismatch: %q vs %q", got, data)
	}
}

func TestInstallerCleanupOldBackups(t *testing.T) {
	dir := t.TempDir()
	ins := &Installer{BackupDir: dir}

	// create an old backup
	oldFile := filepath.Join(dir, "old.bak")
	os.WriteFile(oldFile, []byte("old"), 0644)
	// make it look very old by modifying time
	veryOld := time.Now().AddDate(0, 0, -30)
	os.Chtimes(oldFile, veryOld, veryOld)

	// create a recent backup
	recentFile := filepath.Join(dir, "recent.bak")
	os.WriteFile(recentFile, []byte("recent"), 0644)

	if err := ins.CleanupOldBackups(7); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatal("expected old backup to be removed")
	}
	if _, err := os.Stat(recentFile); err != nil {
		t.Fatal("expected recent backup to remain")
	}
}

func TestInstallerRollback(t *testing.T) {
	dir := t.TempDir()
	ins := &Installer{BackupDir: dir}
	current := filepath.Join(dir, "current")
	backup := filepath.Join(dir, "backup")
	os.WriteFile(current, []byte("bad"), 0644)
	os.WriteFile(backup, []byte("good"), 0644)

	if err := ins.Rollback(current, backup); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(current)
	if string(got) != "good" {
		t.Fatalf("rollback failed: %q", got)
	}
}

func TestInstallerRollbackMissing(t *testing.T) {
	ins := &Installer{BackupDir: t.TempDir()}
	err := ins.Rollback("/tmp/current", "/tmp/nonexistent")
	if err == nil {
		t.Fatal("expected error for missing backup")
	}
}

func TestNewInstaller(t *testing.T) {
	ins := NewInstaller()
	if ins.BackupDir == "" {
		t.Fatal("expected BackupDir")
	}
}


