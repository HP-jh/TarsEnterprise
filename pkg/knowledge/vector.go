package knowledge

import (
	"sort"
	"sync"
)

// VectorStore 内存向量存储
type VectorStore struct {
	chunks map[string]Chunk // chunkID -> Chunk
	mu     sync.RWMutex
}

func NewVectorStore() *VectorStore {
	return &VectorStore{chunks: make(map[string]Chunk)}
}

// Add 添加带向量的 chunk
func (vs *VectorStore) Add(c Chunk) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.chunks[c.ID] = c
}

// RemoveDoc 移除某文档的所有 chunk
func (vs *VectorStore) RemoveDoc(docID string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	for id, c := range vs.chunks {
		if c.DocID == docID {
			delete(vs.chunks, id)
		}
	}
}

// Search 向量相似度检索
func (vs *VectorStore) Search(queryVec []float32, topK int) []SearchResult {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	var results []SearchResult
	for _, c := range vs.chunks {
		if len(c.Vector) == 0 || len(c.Vector) != len(queryVec) {
			continue
		}
		score := CosineSimilarity(queryVec, c.Vector)
		if score > 0 {
			results = append(results, SearchResult{
				Chunk: c,
				Score: score,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if topK > 0 && topK < len(results) {
		results = results[:topK]
	}
	return results
}

// Count 返回存储的 chunk 数量
func (vs *VectorStore) Count() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return len(vs.chunks)
}
