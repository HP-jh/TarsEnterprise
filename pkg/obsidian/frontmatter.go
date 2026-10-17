package obsidian

import (
	"fmt"
	"strings"
)

// extractFrontmatter 解析 YAML frontmatter
func extractFrontmatter(content string) (map[string]interface{}, string) {
	if !strings.HasPrefix(content, "---") {
		return nil, content
	}
	endIdx := strings.Index(content[3:], "\n---")
	if endIdx < 0 {
		return nil, content
	}
	endIdx += 3 // account for the initial "---"
	fmStr := content[3:endIdx]
	body := strings.TrimSpace(content[endIdx+4:])

	fm := make(map[string]interface{})
	lines := strings.Split(fmStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			fm[key] = val
		}
	}
	return fm, body
}

// BuildFrontmatter 从 map 构建 frontmatter 字符串
func BuildFrontmatter(fm map[string]interface{}) string {
	if len(fm) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	for k, v := range fm {
		sb.WriteString(fmt.Sprintf("%s: %v\n", k, v))
	}
	sb.WriteString("---\n")
	return sb.String()
}

// extractTags 提取 #tag 和 frontmatter tags
func extractTags(content string) []string {
	seen := make(map[string]bool)
	var tags []string
	words := strings.Fields(content)
	for _, w := range words {
		if strings.HasPrefix(w, "#") && len(w) > 1 {
			tag := strings.TrimPrefix(w, "#")
			tag = strings.TrimFunc(tag, func(r rune) bool {
				return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-')
			})
			if tag != "" && !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

// extractLinks 提取 [[WikiLinks]]
func extractLinks(content string) (backlinks []string, outbound []string) {
	seen := make(map[string]bool)
	for i := 0; i < len(content)-3; i++ {
		if content[i] == '[' && content[i+1] == '[' {
			end := strings.Index(content[i+2:], "]]")
			if end < 0 {
				continue
			}
			link := strings.TrimSpace(content[i+2 : i+2+end])
			if link != "" && !seen[link] {
				seen[link] = true
				outbound = append(outbound, link)
			}
			i += end + 3
		}
	}
	return
}
