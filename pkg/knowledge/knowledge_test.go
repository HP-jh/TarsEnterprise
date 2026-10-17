package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChunker_SplitMarkdown(t *testing.T) {
	c := NewChunker(100, 20)
	content := "# Heading 1\nSome text here.\n## Heading 2\nMore text.\n"
	chunks := c.SplitMarkdown("doc1", content)
	if len(chunks) == 0 {
		t.Fatal("no chunks")
	}
	for i, ck := range chunks {
		if ck.ID == "" {
			t.Errorf("chunk %d missing id", i)
		}
		if ck.DocID != "doc1" {
			t.Errorf("chunk %d wrong doc id", i)
		}
	}
}

func TestEmbedder_Embed(t *testing.T) {
	e := NewEmbedder(64)
	v1 := e.Embed("hello world knowledge base")
	v2 := e.Embed("hello world knowledge base")
	if len(v1) != 64 || len(v2) != 64 {
		t.Error("embedding dim mismatch")
	}
	// Same text -> same embedding (deterministic)
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Error("same text should produce same embedding")
			break
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	if CosineSimilarity(a, b) != 1.0 {
		t.Error("identical vectors should have similarity 1")
	}
	c := []float32{0, 1, 0}
	if CosineSimilarity(a, c) != 0 {
		t.Error("orthogonal vectors should have similarity 0")
	}
}

func TestVectorStore_Search(t *testing.T) {
	vs := NewVectorStore()
	vs.Add(Chunk{ID: "c1", DocID: "d1", Text: "hello world", Vector: []float32{1, 0, 0}})
	vs.Add(Chunk{ID: "c2", DocID: "d1", Text: "goodbye world", Vector: []float32{0, 1, 0}})

	results := vs.Search([]float32{1, 0, 0}, 1)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Chunk.ID != "c1" {
		t.Error("expected c1 first")
	}
}

func TestBase_AddAndSearch(t *testing.T) {
	kb := NewBase("test", "Test KB", "tech")
	doc := &Document{
		Title:   "Test Doc",
		Content: "This is a knowledge base document about AI and MCP.",
		Source:  "test.md",
	}
	if err := kb.AddDocument(doc); err != nil {
		t.Fatal(err)
	}
	if len(kb.ListDocuments()) != 1 {
		t.Error("expected 1 document")
	}

	results, err := kb.Search("AI MCP", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected search results")
	}
}

func TestBase_ImportFromDir(t *testing.T) {
	// Create temp dir with a markdown file
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "note.md"), []byte("# Hello\nWorld"), 0644)

	kb := NewBase("import-test", "Import Test", "tech")
	if err := kb.ImportFromDir(tmpDir); err != nil {
		t.Fatal(err)
	}
	if len(kb.ListDocuments()) != 1 {
		t.Errorf("expected 1 doc, got %d", len(kb.ListDocuments()))
	}
}

func TestManager_InitCompanyBase(t *testing.T) {
	m := NewManager()
	kb := m.InitCompanyBase("tech")
	if kb == nil {
		t.Fatal("nil kb")
	}
	if kb.CompanyType != "tech" {
		t.Error("wrong company type")
	}
}
