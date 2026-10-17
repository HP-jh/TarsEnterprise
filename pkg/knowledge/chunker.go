package knowledge

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Chunker 将文档分块
type Chunker struct {
	ChunkSize    int // 每块最大字符数
	ChunkOverlap int // 块间重叠字符数
}

func NewChunker(size, overlap int) *Chunker {
	if size <= 0 {
		size = 512
	}
	if overlap < 0 || overlap >= size {
		overlap = size / 4
	}
	return &Chunker{ChunkSize: size, ChunkOverlap: overlap}
}

// SplitMarkdown 按 Markdown 结构分块（标题优先分割，次按段落）
func (c *Chunker) SplitMarkdown(docID, content string) []Chunk {
	lines := strings.Split(content, "\n")
	var chunks []Chunk
	var current strings.Builder
	var currentStart int
	var chunkIndex int

	flush := func(end int) {
		text := strings.TrimSpace(current.String())
		if text == "" {
			return
		}
		chunks = append(chunks, Chunk{
			ID:    fmt.Sprintf("%s-c%d", docID, chunkIndex),
			DocID: docID,
			Text:  text,
			Start: currentStart,
			End:   end,
			Meta:  map[string]string{},
		})
		chunkIndex++
		current.Reset()
	}

	byteOffset := 0
	for _, line := range lines {
		lineBytes := len(line) + 1 // +1 for newline
		trimmed := strings.TrimSpace(line)

		// Heading boundary: flush before new heading
		if strings.HasPrefix(trimmed, "#") {
			flush(byteOffset)
			currentStart = byteOffset
		}

		current.WriteString(line)
		current.WriteByte('\n')

		// Size boundary
		if current.Len() >= c.ChunkSize {
			flush(byteOffset + lineBytes)
			currentStart = byteOffset + lineBytes - c.ChunkOverlap
			if currentStart < 0 {
				currentStart = 0
			}
			// Carry overlap
			full := current.String()
			overlapStart := utf8.RuneCountInString(full) - c.ChunkOverlap
			if overlapStart < 0 {
				overlapStart = 0
			}
			runes := []rune(full)
			if overlapStart < len(runes) {
				current.Reset()
				current.WriteString(string(runes[overlapStart:]))
			}
		}

		byteOffset += lineBytes
	}
	flush(byteOffset)
	return chunks
}

// SplitText 纯文本分块
func (c *Chunker) SplitText(docID, content string) []Chunk {
	var chunks []Chunk
	var chunkIndex int
	runes := []rune(content)
	for i := 0; i < len(runes); i += c.ChunkSize - c.ChunkOverlap {
		end := i + c.ChunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, Chunk{
			ID:    fmt.Sprintf("%s-c%d", docID, chunkIndex),
			DocID: docID,
			Text:  string(runes[i:end]),
			Start: i,
			End:   end,
			Meta:  map[string]string{},
		})
		chunkIndex++
		if end == len(runes) {
			break
		}
	}
	return chunks
}
