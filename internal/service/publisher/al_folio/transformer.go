package al_folio

import (
	"context"
	"fmt"
	"github.com/ifuryst/ripple/pkg/util"
	"strings"
	"time"
)

// AlFolioTransformer converts Notion content to Al-Folio-compatible Markdown
type AlFolioTransformer struct {
	baseTransformer *MarkdownTransformer
}

func NewAlFolioTransformer() *AlFolioTransformer {
	return &AlFolioTransformer{
		baseTransformer: NewMarkdownTransformer(),
	}
}

func (t *AlFolioTransformer) Transform(ctx context.Context, content string, metadata map[string]string) (string, error) {
	// Convert Notion blocks JSON to markdown
	markdownContent, err := convertNotionBlocksToMarkdown(content)
	if err != nil {
		return "", fmt.Errorf("notion blocks to markdown conversion failed: %w", err)
	}

	// Generate Al-Folio-specific front matter
	frontMatter := t.generateAlFolioFrontMatter(metadata)

	return frontMatter + "\n\n" + markdownContent, nil
}

func (t *AlFolioTransformer) generateAlFolioFrontMatter(metadata map[string]string) string {
	var frontMatter []string
	frontMatter = append(frontMatter, "---")

	// Required fields
	frontMatter = append(frontMatter, "layout: post")

	// Title
	if title := metadata["title"]; title != "" {
		frontMatter = append(frontMatter, fmt.Sprintf("title: \"%s\"", util.EscapeYAML(title)))
	}

	// Date - format for Al-Folio
	if dateStr := metadata["publish_date"]; dateStr != "" {
		// Try to parse the date and format it correctly
		if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
			// Format as Al-Folio expects: YYYY-MM-DDTHH:MM:SS+08:00
			formattedDate := date.Format("2006-01-02T15:04:05-07:00")
			frontMatter = append(frontMatter, fmt.Sprintf("date: %s", formattedDate))
		}
	} else {
		// Use current time if no date provided
		now := time.Now()
		formattedDate := now.Format("2006-01-02T15:04:05-07:00")
		frontMatter = append(frontMatter, fmt.Sprintf("date: %s", formattedDate))
	}

	// Tags - can be multiple, space-separated or array format
	if tags := metadata["tags"]; tags != "" {
		// Parse tags from various formats
		tagList := util.ParseTags(tags)
		if len(tagList) > 0 {
			if len(tagList) == 1 {
				frontMatter = append(frontMatter, fmt.Sprintf("tags: %s", tagList[0]))
			} else {
				frontMatter = append(frontMatter, "tags:")
				for _, tag := range tagList {
					frontMatter = append(frontMatter, fmt.Sprintf("  - %s", tag))
				}
			}
		}
	}

	// Categories - similar to tags
	if categories := metadata["categories"]; categories != "" {
		categoryList := util.ParseTags(categories) // Same parsing logic
		if len(categoryList) > 0 {
			if len(categoryList) == 1 {
				frontMatter = append(frontMatter, fmt.Sprintf("categories: %s", categoryList[0]))
			} else {
				frontMatter = append(frontMatter, "categories:")
				for _, category := range categoryList {
					frontMatter = append(frontMatter, fmt.Sprintf("  - %s", category))
				}
			}
		}
	}

	// Al-Folio-specific settings
	frontMatter = append(frontMatter, "giscus_comments: true")
	frontMatter = append(frontMatter, "tabs: true")
	frontMatter = append(frontMatter, "pretty_table: true")

	// Check if we need TOC (Table of Contents)
	if t.shouldAddTOC(metadata) {
		frontMatter = append(frontMatter, "toc:")
		frontMatter = append(frontMatter, "  sidebar: left")
	}

	frontMatter = append(frontMatter, "---")

	return strings.Join(frontMatter, "\n")
}

func (t *AlFolioTransformer) shouldAddTOC(metadata map[string]string) bool {
	// Add TOC if the content is long enough or has headers
	// This is a simple heuristic - you can make it more sophisticated

	// Check if TOC is explicitly requested
	if toc := metadata["toc"]; toc == "true" || toc == "yes" {
		return true
	}

	// Check content length or other factors
	if content := metadata["content"]; content != "" {
		// Count headers in content
		headerCount := strings.Count(content, "#")
		if headerCount >= 3 {
			return true
		}

		// Check content length
		if len(content) > 2000 {
			return true
		}
	}

	return false
}
