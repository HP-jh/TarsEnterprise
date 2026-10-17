package obsidian

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/HP-jh/TarsEnterprise/pkg/knowledge"
)

// SyncEngine Obsidian ↔ KnowledgeBase 双向同步引擎
type SyncEngine struct {
	VaultPath    string
	KnowledgeBase *knowledge.Base
	Interval     time.Duration
	mu           sync.Mutex
	running      bool
	stopCh       chan struct{}
}

func NewSyncEngine(vaultPath string, kb *knowledge.Base) *SyncEngine {
	return &SyncEngine{
		VaultPath:     vaultPath,
		KnowledgeBase: kb,
		Interval:      30 * time.Second,
		stopCh:        make(chan struct{}),
	}
}

// SyncOnce 执行一次全量同步（Vault → KnowledgeBase）
func (se *SyncEngine) SyncOnce() error {
	vault, err := OpenVault(se.VaultPath)
	if err != nil {
		return err
	}
	for _, note := range vault.Notes {
		doc := &knowledge.Document{
			Title:   note.Title,
			Content: note.Content,
			Source:  note.Path,
			Tags:    note.Tags,
			Meta:    map[string]string{},
		}
		for k, v := range note.Frontmatter {
			doc.Meta[k] = fmt.Sprintf("%v", v)
		}
		if err := se.KnowledgeBase.AddDocument(doc); err != nil {
			return err
		}
	}
	return nil
}

// StartPolling 启动轮询同步（模拟 fsnotify，v0.2.0 用轮询）
func (se *SyncEngine) StartPolling(ctx context.Context) {
	se.mu.Lock()
	if se.running {
		se.mu.Unlock()
		return
	}
	se.running = true
	se.mu.Unlock()

	ticker := time.NewTicker(se.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-se.stopCh:
			return
		case <-ticker.C:
			_ = se.SyncOnce()
		}
	}
}

// Stop 停止轮询
func (se *SyncEngine) Stop() {
	se.mu.Lock()
	defer se.mu.Unlock()
	if se.running {
		close(se.stopCh)
		se.running = false
	}
}

// WriteBack 将 KnowledgeBase 中的文档写回 Obsidian vault
func (se *SyncEngine) WriteBack(doc *knowledge.Document) error {
	path := doc.Source
	if path == "" {
		path = filepath.Join(se.VaultPath, doc.Title+".md")
	}

	fm := make(map[string]interface{})
	for k, v := range doc.Meta {
		fm[k] = v
	}
	if len(doc.Tags) > 0 {
		fm["tags"] = strings.Join(doc.Tags, ", ")
	}
	fm["source"] = "tarsenterprise"
	fm["updated"] = time.Now().Format(time.RFC3339)

	note := &Note{
		Path:        path,
		Title:       doc.Title,
		Content:     doc.Content,
		Frontmatter: fm,
	}
	return WriteNote(note)
}

// ConnectorConfig Obsidian 连接配置
type ConnectorConfig struct {
	VaultPath      string   `json:"vault_path"`
	AutoSync       bool     `json:"auto_sync"`
	SyncInterval   int      `json:"sync_interval_seconds"`
	IncludeTags    []string `json:"include_tags"`
	ExcludeFolders []string `json:"exclude_folders"`
}

func DefaultConnectorConfig() *ConnectorConfig {
	return &ConnectorConfig{
		VaultPath:      "./obsidian-vault",
		AutoSync:       true,
		SyncInterval:   30,
		IncludeTags:    []string{},
		ExcludeFolders: []string{".obsidian", "_templates", "_attachments"},
	}
}
