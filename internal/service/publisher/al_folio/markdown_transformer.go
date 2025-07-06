package al_folio

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// MarkdownTransformer converts various formats to Markdown for Al-Folio
type MarkdownTransformer struct{}

func NewMarkdownTransformer() *MarkdownTransformer {
	return &MarkdownTransformer{}
}

func (t *MarkdownTransformer) Transform(ctx context.Context, content string, metadata map[string]string) (string, error) {
	// Basic transformation from Notion-style content to Markdown
	// This is a simplified version - you might want to use a proper Notion-to-Markdown library

	transformed := content

	// Convert basic formatting
	transformed = t.convertHeaders(transformed)
	transformed = t.convertLists(transformed)
	transformed = t.convertLinks(transformed)
	transformed = t.convertEmphasis(transformed)
	transformed = t.convertCodeBlocks(transformed)

	// Add frontmatter if metadata is provided
	if len(metadata) > 0 {
		frontmatter := t.generateFrontmatter(metadata)
		transformed = frontmatter + "\n\n" + transformed
	}

	return transformed, nil
}

func (t *MarkdownTransformer) convertHeaders(content string) string {
	// Convert headers (this is simplified - real implementation would be more complex)
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			result = append(result, line)
			continue
		}

		// Check for header patterns (simplified)
		if strings.HasPrefix(line, "# ") {
			result = append(result, line)
		} else if isLikelyHeader(line) {
			result = append(result, "## "+line)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func (t *MarkdownTransformer) convertLists(content string) string {
	// Convert bullet points and numbered lists
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Convert bullet points
		if strings.HasPrefix(line, "• ") {
			line = "- " + strings.TrimPrefix(line, "• ")
		} else if strings.HasPrefix(line, "◦ ") {
			line = "  - " + strings.TrimPrefix(line, "◦ ")
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func (t *MarkdownTransformer) convertLinks(content string) string {
	// Convert links to markdown format
	// This is a basic implementation
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	return linkRegex.ReplaceAllString(content, "[$1]($2)")
}

func (t *MarkdownTransformer) convertEmphasis(content string) string {
	// Convert bold and italic formatting
	content = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(content, "**$1**")
	content = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(content, "*$1*")
	return content
}

func (t *MarkdownTransformer) convertCodeBlocks(content string) string {
	// Convert code blocks
	content = regexp.MustCompile("```([^`]+)```").ReplaceAllString(content, "```\n$1\n```")
	content = regexp.MustCompile("`([^`]+)`").ReplaceAllString(content, "`$1`")
	return content
}

func (t *MarkdownTransformer) generateFrontmatter(metadata map[string]string) string {
	var frontmatter []string
	frontmatter = append(frontmatter, "---")

	// Add common frontmatter fields
	if title := metadata["title"]; title != "" {
		frontmatter = append(frontmatter, fmt.Sprintf("title: \"%s\"", title))
	}

	if date := metadata["publish_date"]; date != "" {
		frontmatter = append(frontmatter, fmt.Sprintf("date: %s", date))
	}

	if author := metadata["author"]; author != "" {
		frontmatter = append(frontmatter, fmt.Sprintf("author: \"%s\"", author))
	}

	if tags := metadata["tags"]; tags != "" {
		frontmatter = append(frontmatter, fmt.Sprintf("tags: [%s]", tags))
	}

	if summary := metadata["summary"]; summary != "" {
		frontmatter = append(frontmatter, fmt.Sprintf("description: \"%s\"", summary))
	}

	// Add custom metadata
	for key, value := range metadata {
		if !isCommonField(key) {
			frontmatter = append(frontmatter, fmt.Sprintf("%s: \"%s\"", key, value))
		}
	}

	frontmatter = append(frontmatter, "---")
	return strings.Join(frontmatter, "\n")
}

func isLikelyHeader(line string) bool {
	// Simple heuristic to detect headers
	return len(line) < 100 && !strings.Contains(line, ".")
}

func isCommonField(key string) bool {
	commonFields := []string{"title", "publish_date", "author", "tags", "summary"}
	for _, field := range commonFields {
		if key == field {
			return true
		}
	}
	return false
}