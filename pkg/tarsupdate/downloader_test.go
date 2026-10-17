package update

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSHA256SUMS(t *testing.T) {
	content := `
# SHA256SUMS
abc123  file1
# comment
def456 *file2
`
	m := ParseSHA256SUMS(content)
	if m["file1"] != "abc123" {
		t.Fatalf("file1: got %q", m["file1"])
	}
	if m["file2"] != "def456" {
		t.Fatalf("file2: got %q", m["file2"])
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
}

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	data := []byte("hello tars-update")
	os.WriteFile(path, data, 0644)

	h := sha256.New()
	h.Write(data)
	expected := fmt.Sprintf("%x", h.Sum(nil))

	if err := VerifySHA256(path, expected); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(path, "badhash"); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestVerifySHA256MissingFile(t *testing.T) {
	err := VerifySHA256("/nonexistent/path", "abc")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestNewDownloader(t *testing.T) {
	d := NewDownloader()
	if d.HTTPClient == nil {
		t.Fatal("expected HTTPClient")
	}
}
