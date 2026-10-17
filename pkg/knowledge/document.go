package knowledge

import (
	"time"
)

// Document 知识库文档
type Document struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	Source    string            `json:"source"`    // file path or URL
	Tags      []string          `json:"tags"`
	Meta      map[string]string `json:"meta"`
	Chunks    []Chunk           `json:"-"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Chunk 文档分块
type Chunk struct {
	ID       string    `json:"id"`
	DocID    string    `json:"doc_id"`
	Text     string    `json:"text"`
	Start    int       `json:"start"`    // byte offset in original doc
	End      int       `json:"end"`
	Vector   []float32 `json:"-"`
	Meta     map[string]string `json:"meta"`
}

// SearchResult 检索结果
type SearchResult struct {
	Chunk      Chunk   `json:"chunk"`
	Score      float64 `json:"score"`
	DocTitle   string  `json:"doc_title"`
	DocSource  string  `json:"doc_source"`
}
