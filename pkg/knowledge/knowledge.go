package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Base 知识库
type Base struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	CompanyType string            `json:"company_type"` // education/tech/kpi/...
	Embedder    *Embedder
	Chunker     *Chunker
	Store       *VectorStore
	docs        map[string]*Document
	mu          sync.RWMutex
}

func NewBase(id, name, companyType string) *Base {
	return &Base{
		ID:          id,
		Name:        name,
		CompanyType: companyType,
		Embedder:    NewEmbedder(128),
		Chunker:     NewChunker(512, 128),
		Store:       NewVectorStore(),
		docs:        make(map[string]*Document),
	}
}

// AddDocument 添加文档到知识库
func (kb *Base) AddDocument(doc *Document) error {
	if doc.ID == "" {
		doc.ID = fmt.Sprintf("doc-%d", time.Now().UnixNano())
	}
	doc.UpdatedAt = time.Now()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = doc.UpdatedAt
	}

	// Chunk
	var chunks []Chunk
	if strings.HasSuffix(doc.Source, ".md") || strings.Contains(doc.Content, "#") {
		chunks = kb.Chunker.SplitMarkdown(doc.ID, doc.Content)
	} else {
		chunks = kb.Chunker.SplitText(doc.ID, doc.Content)
	}

	// Embed
	for i := range chunks {
		chunks[i].Vector = kb.Embedder.Embed(chunks[i].Text)
	}

	doc.Chunks = chunks

	kb.mu.Lock()
	defer kb.mu.Unlock()

	// Remove old chunks if exists
	kb.Store.RemoveDoc(doc.ID)

	// Add new chunks
	for i := range chunks {
		kb.Store.Add(chunks[i])
	}
	kb.docs[doc.ID] = doc
	return nil
}

// RemoveDocument 移除文档
func (kb *Base) RemoveDocument(docID string) {
	kb.mu.Lock()
	defer kb.mu.Unlock()
	kb.Store.RemoveDoc(docID)
	delete(kb.docs, docID)
}

// Search 检索知识库
func (kb *Base) Search(query string, topK int) ([]SearchResult, error) {
	queryVec := kb.Embedder.Embed(query)
	results := kb.Store.Search(queryVec, topK)

	kb.mu.RLock()
	defer kb.mu.RUnlock()

	// Enrich with doc info
	for i := range results {
		if doc, ok := kb.docs[results[i].Chunk.DocID]; ok {
			results[i].DocTitle = doc.Title
			results[i].DocSource = doc.Source
		}
	}
	return results, nil
}

// ListDocuments 列出所有文档
func (kb *Base) ListDocuments() []*Document {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	var out []*Document
	for _, d := range kb.docs {
		out = append(out, d)
	}
	return out
}

// ImportFromDir 从目录批量导入 Markdown 文件
func (kb *Base) ImportFromDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		title := strings.TrimSuffix(filepath.Base(path), ".md")
		doc := &Document{
			Title:   title,
			Content: string(data),
			Source:  path,
			Tags:    []string{},
			Meta:    map[string]string{},
		}
		return kb.AddDocument(doc)
	})
}

// Manager 知识库管理器（支持多库）
type Manager struct {
	bases map[string]*Base
	mu    sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{bases: make(map[string]*Base)}
}

func (m *Manager) CreateBase(id, name, companyType string) *Base {
	m.mu.Lock()
	defer m.mu.Unlock()
	kb := NewBase(id, name, companyType)
	m.bases[id] = kb
	return kb
}

func (m *Manager) GetBase(id string) (*Base, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	kb, ok := m.bases[id]
	return kb, ok
}

func (m *Manager) ListBases() []*Base {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Base
	for _, kb := range m.bases {
		out = append(out, kb)
	}
	return out
}

// InitCompanyBase 为指定公司类型初始化预置知识库
func (m *Manager) InitCompanyBase(companyType string) *Base {
	id := fmt.Sprintf("kb-%s", companyType)
	if kb, ok := m.GetBase(id); ok {
		return kb
	}
	kb := m.CreateBase(id, fmt.Sprintf("%s Knowledge Base", companyType), companyType)
	// Pre-load templates if templates dir exists
	tmplDir := filepath.Join("templates", companyType)
	_ = kb.ImportFromDir(tmplDir)
	return kb
}
