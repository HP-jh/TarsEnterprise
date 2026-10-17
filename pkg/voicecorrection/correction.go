package voicecorrection

import (
	"context"
	"strings"
	"sync"
)

// Corrector 语音纠错 MCP 工具（v0.2.0 stub，待 B39 TarsVoiceCorrection 完成后对接）
type Corrector struct {
	// B39 服务地址
	ServiceURL string `json:"service_url"`
	enabled    bool
	mu         sync.RWMutex
}

func NewCorrector(serviceURL string) *Corrector {
	return &Corrector{
		ServiceURL: serviceURL,
		enabled:    serviceURL != "",
	}
}

// Correct 执行语音文本纠错
func (c *Corrector) Correct(ctx context.Context, text string) (*Result, error) {
	c.mu.RLock()
	enabled := c.enabled
	c.mu.RUnlock()

	if !enabled {
		// v0.2.0 fallback: 本地规则纠错
		return c.localCorrect(text), nil
	}

	// TODO: v0.3.0 调用 B39 TarsVoiceCorrection 服务
	return c.localCorrect(text), nil
}

// Result 纠错结果
type Result struct {
	Original   string   `json:"original"`
	Corrected  string   `json:"corrected"`
	Changes    []Change `json:"changes"`
	Confidence float64  `json:"confidence"`
	Source     string   `json:"source"` // "local" or "b39"
}

type Change struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// localCorrect 本地规则纠错（常见语音输入错误）
func (c *Corrector) localCorrect(text string) *Result {
	corrected := text
	var changes []Change

	// Common phonetic confusion pairs (Chinese-centric + some English)
	replacements := []struct {
		from string
		to   string
	}{
		{"知识库", "知识库"}, // self-mapping for identity check
		{"网关", "网关"},
		{"配置", "配置"},
		{"沙箱", "沙箱"},
		{"智能体", "智能体"},
		{"openclaw", "OpenClaw"},
		{"obsidian", "Obsidian"},
		{"mcp", "MCP"},
		{"ai", "AI"},
		{"api", "API"},
	}

	for _, r := range replacements {
		idx := strings.Index(strings.ToLower(corrected), strings.ToLower(r.from))
		if idx >= 0 && corrected[idx:idx+len(r.from)] != r.to {
			original := corrected[idx : idx+len(r.from)]
			corrected = corrected[:idx] + r.to + corrected[idx+len(r.from):]
			changes = append(changes, Change{
				From:  original,
				To:    r.to,
				Start: idx,
				End:   idx + len(r.to),
			})
		}
	}

	return &Result{
		Original:   text,
		Corrected:  corrected,
		Changes:    changes,
		Confidence: 0.7,
		Source:     "local",
	}
}

// AsMCPTool 将纠错器包装为 MCP Tool 格式
func (c *Corrector) AsMCPTool() map[string]interface{} {
	return map[string]interface{}{
		"name":        "voice_correction",
		"description": "Correct voice input text for common phonetic errors and auto-capitalize tech terms",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "The voice-recognized text to correct",
				},
			},
			"required": []string{"text"},
		},
	}
}

// SetServiceURL 更新 B39 服务地址
func (c *Corrector) SetServiceURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ServiceURL = url
	c.enabled = url != ""
}

// Enabled 是否启用
func (c *Corrector) Enabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}
