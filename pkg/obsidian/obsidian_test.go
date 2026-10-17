package obsidian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HP-jh/TarsEnterprise/pkg/knowledge"
)

func TestExtractFrontmatter(t *testing.T) {
	content := "---\ntitle: Hello\ntags: a, b\n---\n\n# Body\nText"
	fm, body := extractFrontmatter(content)
	if fm == nil {
		t.Fatal("nil frontmatter")
	}
	if fm["title"] != "Hello" {
		t.Errorf("title = %v", fm["title"])
	}
	if body != "# Body\nText" {
		t.Errorf("body = %q", body)
	}
}

func TestExtractFrontmatter_NoFrontmatter(t *testing.T) {
	content := "# Just body"
	fm, body := extractFrontmatter(content)
	if fm != nil {
		t.Error("expected nil frontmatter")
	}
	if body != "# Just body" {
		t.Error("body mismatch")
	}
}

func TestExtractTags(t *testing.T) {
	tags := extractTags("Hello #world #AI test")
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestExtractLinks(t *testing.T) {
	_, outbound := extractLinks("See [[Project A]] and [[Project B]]")
	if len(outbound) != 2 {
		t.Errorf("expected 2 links, got %d", len(outbound))
	}
}

func TestOpenVault(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "note1.md"), []byte("# Note 1\nContent"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "note2.md"), []byte("---\ntag: test\n---\n# Note 2"), 0644)

	vault, err := OpenVault(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vault.Notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(vault.Notes))
	}
}

func TestWriteNote(t *testing.T) {
	tmpDir := t.TempDir()
	note := &Note{
		Path: filepath.Join(tmpDir, "out.md"),
		Title: "Out",
		Content: "Body text",
		Frontmatter: map[string]interface{}{"type": "test"},
	}
	if err := WriteNote(note); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(note.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("empty file")
	}
}

func TestSyncEngine_SyncOnce(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "sync.md"), []byte("# Sync Test\nContent here"), 0644)

	kb := knowledge.NewBase("obs-test", "Obsidian Test", "tech")
	se := NewSyncEngine(tmpDir, kb)
	if err := se.SyncOnce(); err != nil {
		t.Fatal(err)
	}
	if len(kb.ListDocuments()) != 1 {
		t.Errorf("expected 1 doc in kb, got %d", len(kb.ListDocuments()))
	}
}

func TestSyncEngine_WriteBack(t *testing.T) {
	tmpDir := t.TempDir()
	kb := knowledge.NewBase("wb-test", "WriteBack Test", "tech")
	se := NewSyncEngine(tmpDir, kb)

	doc := &knowledge.Document{
		Title:   "WriteBack",
		Content: "# Hello\nWorld",
		Tags:    []string{"test"},
		Meta:    map[string]string{"author": "TE"},
	}
	if err := se.WriteBack(doc); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(tmpDir, "WriteBack.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("empty file")
	}
}
