package knowledge

import (
	"math"
	"strings"
	"unicode"
)

// Embedder 文本向量化器（v0.2.0 使用统计词频 + TF-IDF 风格，不依赖外部模型）
type Embedder struct {
	dim      int
	vocab    map[string]int // word -> index
	vocabSeq int
}

func NewEmbedder(dim int) *Embedder {
	if dim <= 0 {
		dim = 128
	}
	return &Embedder{
		dim:   dim,
		vocab: make(map[string]int),
	}
}

// Embed 将文本编码为向量
func (e *Embedder) Embed(text string) []float32 {
	vec := make([]float32, e.dim)
	words := tokenize(text)
	if len(words) == 0 {
		return vec
	}

	// Term frequency
	tf := make(map[string]float64)
	for _, w := range words {
		tf[w]++
	}
	for w := range tf {
		tf[w] /= float64(len(words))
	}

	// Build vocab on the fly
	for w := range tf {
		if _, ok := e.vocab[w]; !ok {
			e.vocab[w] = e.vocabSeq % e.dim
			e.vocabSeq++
		}
	}

	// Accumulate into vector slots
	for w, freq := range tf {
		idx := e.vocab[w]
		// Simple hash-based weight to spread values
		hash := fnvHash(w)
		weight := freq * (1.0 + 0.5*math.Sin(float64(hash)))
		vec[idx] += float32(weight)
	}

	// L2 normalize
	norm := float64(0)
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vec {
			vec[i] /= float32(norm)
		}
	}
	return vec
}

// CosineSimilarity 计算两个向量的余弦相似度
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func tokenize(text string) []string {
	var words []string
	lower := strings.ToLower(text)
	// Simple tokenization: split by non-letter/digit
	var current strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 2 {
				words = append(words, current.String())
			}
			current.Reset()
		}
	}
	if current.Len() > 2 {
		words = append(words, current.String())
	}
	return words
}

func fnvHash(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
