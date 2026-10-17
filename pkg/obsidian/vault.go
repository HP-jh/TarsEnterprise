package obsidian

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Vault 表示一个 Obsidian vault
type Vault struct {
	Path    string   `json:"path"`
	Name    string   `json:"name"`
	Notes   []Note   `json:"notes"`
}

// Note Obsidian 笔记
type Note struct {
	Path       string            `json:"path"`
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	Frontmatter map[string]interface{} `json:"frontmatter"`
	Tags       []string          `json:"tags"`
	Backlinks  []string          `json:"backlinks"`
	Outbound   []string          `json:"outbound_links"`
}

// OpenVault 打开 Obsidian vault 目录
func OpenVault(vaultPath string) (*Vault, error) {
	info, err := os.Stat(vaultPath)
	if err != nil {
		return nil, fmt.Errorf("vault path not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("vault path is not a directory")
	}

	vault := &Vault{
		Path: vaultPath,
		Name: filepath.Base(vaultPath),
	}

	err = filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.HasSuffix(path, ".md") {
			note, err := parseNote(path)
			if err == nil {
				vault.Notes = append(vault.Notes, *note)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return vault, nil
}

func parseNote(path string) (*Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	fm, body := extractFrontmatter(content)
	title := strings.TrimSuffix(filepath.Base(path), ".md")
	tags := extractTags(body)
	backlinks, outbound := extractLinks(body)

	return &Note{
		Path:        path,
		Title:       title,
		Content:     body,
		Frontmatter: fm,
		Tags:        tags,
		Backlinks:   backlinks,
		Outbound:    outbound,
	}, nil
}

// WriteNote 写回笔记到 vault
func WriteNote(note *Note) error {
	var sb strings.Builder
	if len(note.Frontmatter) > 0 {
		sb.WriteString("---\n")
		for k, v := range note.Frontmatter {
			sb.WriteString(fmt.Sprintf("%s: %v\n", k, v))
		}
		sb.WriteString("---\n\n")
	}
	sb.WriteString(note.Content)
	return os.WriteFile(note.Path, []byte(sb.String()), 0644)
}
