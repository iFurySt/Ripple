package al_folio

import (
	"encoding/json"
	"fmt"
	"strings"
)

// convertNotionBlocksToMarkdown converts raw Notion blocks JSON to markdown format
func convertNotionBlocksToMarkdown(blocksJSON string) (string, error) {
	var blocks []map[string]any
	if err := json.Unmarshal([]byte(blocksJSON), &blocks); err != nil {
		return "", fmt.Errorf("failed to unmarshal blocks: %w", err)
	}

	// Convert blocks to markdown format
	var content []string
	numberedListCounter := 0

	for i := 0; i < len(blocks); i++ {
		block := blocks[i]
		if blockType, _ := block["type"].(string); blockType == "table" {
			markdown, nextIndex := convertTableToMarkdown(blocks, i)
			if markdown != "" {
				content = append(content, markdown)
			}
			i = nextIndex - 1
			numberedListCounter = 0
			continue
		} else if blockType == "table_row" {
			continue
		}

		markdown, skip, isNumberedList := convertBlockToMarkdownWithCounter(block, &numberedListCounter)
		if skip {
			continue
		}

		// Reset counter if this is not a numbered list item
		if !isNumberedList {
			numberedListCounter = 0
		}

		content = append(content, markdown)
	}

	return strings.Join(content, "\n"), nil
}

func convertTableToMarkdown(blocks []map[string]any, tableIndex int) (string, int) {
	tableContent, _ := blocks[tableIndex]["table"].(map[string]any)
	hasColumnHeader, _ := tableContent["has_column_header"].(bool)
	tableWidth := intFromAny(tableContent["table_width"])

	var rows [][]string
	nextIndex := tableIndex + 1
	for nextIndex < len(blocks) {
		blockType, _ := blocks[nextIndex]["type"].(string)
		if blockType != "table_row" {
			break
		}
		if row := extractTableRowMarkdown(blocks[nextIndex]); len(row) > 0 {
			rows = append(rows, row)
			if len(row) > tableWidth {
				tableWidth = len(row)
			}
		}
		nextIndex++
	}

	if len(rows) == 0 || tableWidth == 0 {
		return "", nextIndex
	}

	for i := range rows {
		for len(rows[i]) < tableWidth {
			rows[i] = append(rows[i], "")
		}
	}

	var out []string
	if hasColumnHeader {
		out = append(out, markdownTableRow(rows[0]))
		out = append(out, markdownSeparatorRow(tableWidth))
		for _, row := range rows[1:] {
			out = append(out, markdownTableRow(row))
		}
	} else {
		headers := make([]string, tableWidth)
		out = append(out, markdownTableRow(headers))
		out = append(out, markdownSeparatorRow(tableWidth))
		for _, row := range rows {
			out = append(out, markdownTableRow(row))
		}
	}

	return strings.Join(out, "\n"), nextIndex
}

func extractTableRowMarkdown(block map[string]any) []string {
	rowContent, ok := block["table_row"].(map[string]any)
	if !ok {
		return nil
	}
	cells, ok := rowContent["cells"].([]any)
	if !ok {
		return nil
	}

	row := make([]string, 0, len(cells))
	for _, cell := range cells {
		richText, _ := cell.([]any)
		row = append(row, escapeMarkdownTableCell(extractRichTextArrayToMarkdown(richText)))
	}
	return row
}

func extractRichTextArrayToMarkdown(richText []any) string {
	var text string
	for _, rt := range richText {
		if rtMap, ok := rt.(map[string]any); ok {
			if plainText, ok := rtMap["plain_text"].(string); ok {
				text += applyRichTextFormatting(plainText, rtMap)
			}
		}
	}
	return cleanText(text)
}

func markdownTableRow(cells []string) string {
	return "| " + strings.Join(cells, " | ") + " |"
}

func markdownSeparatorRow(width int) string {
	parts := make([]string, width)
	for i := range parts {
		parts[i] = "---"
	}
	return "| " + strings.Join(parts, " | ") + " |"
}

func escapeMarkdownTableCell(text string) string {
	text = strings.ReplaceAll(text, "\n", "<br>")
	text = strings.ReplaceAll(text, "|", `\|`)
	return text
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

func convertBlockToMarkdownWithCounter(block map[string]any, numberedListCounter *int) (content string, skip bool, isNumberedList bool) {
	blockType, ok := block["type"].(string)
	if !ok {
		skip = true
		return
	}

	blockContent, ok := block[blockType].(map[string]any)
	if !ok {
		skip = true
		return
	}

	switch blockType {
	case "paragraph":
		text := extractRichTextToMarkdown(blockContent)
		content = cleanText(text)
		return
	case "heading_1":
		text := extractRichTextToMarkdown(blockContent)
		if text != "" {
			content = "# " + cleanText(text)
			return
		}
	case "heading_2":
		text := extractRichTextToMarkdown(blockContent)
		if text != "" {
			content = "## " + cleanText(text)
			return
		}
	case "heading_3":
		text := extractRichTextToMarkdown(blockContent)
		if text != "" {
			content = "### " + cleanText(text)
			return
		}
	case "bulleted_list_item":
		text := extractRichTextToMarkdown(blockContent)
		if text != "" {
			content = "- " + cleanText(text)
			return
		}
	case "numbered_list_item":
		text := extractRichTextToMarkdown(blockContent)
		if text != "" {
			*numberedListCounter++
			content = fmt.Sprintf("%d. %s", *numberedListCounter, cleanText(text))
			isNumberedList = true
			return
		}
	case "quote":
		text := extractRichTextToMarkdown(blockContent)
		if text != "" {
			content = "> " + cleanText(text)
			return
		}
	case "code":
		text := extractRichTextToMarkdown(blockContent)
		language := ""
		if lang, ok := blockContent["language"].(string); ok {
			language = lang
		}
		if text != "" {
			content = "```" + language + "\n" + cleanText(text) + "\n```"
			return
		}
	case "divider":
		content = "---"
		return
	case "image":
		// Handle image blocks
		content = convertImageBlockToMarkdown(blockContent)
		return
	case "column_list":
		// Column lists are container blocks, they don't have content themselves
		// Their content comes from their child column blocks
		content = ""
		return
	case "column":
		// Column blocks are also containers, their content comes from child blocks
		content = ""
		return
	default:
		// For other block types, just extract the text
		text := extractRichTextToMarkdown(blockContent)
		content = cleanText(text)
		return
	}

	return content, false, false
}

// Legacy function for backward compatibility
func convertBlockToMarkdown(block map[string]any) (content string, skip bool) {
	var counter int
	content, skip, _ = convertBlockToMarkdownWithCounter(block, &counter)
	return content, skip
}

// convertImageBlockToMarkdown converts Notion image blocks to Jekyll figure format
func convertImageBlockToMarkdown(blockContent map[string]any) string {
	// Extract image URL from different possible sources
	var imageURL string

	// Try to get from file object (for uploaded images)
	if fileObj, ok := blockContent["file"].(map[string]any); ok {
		if url, ok := fileObj["url"].(string); ok {
			imageURL = url
		}
	}

	// Try to get from external object (for external images)
	if imageURL == "" {
		if externalObj, ok := blockContent["external"].(map[string]any); ok {
			if url, ok := externalObj["url"].(string); ok {
				imageURL = url
			}
		}
	}

	// Return Jekyll figure format directly
	if imageURL != "" {
		return fmt.Sprintf(`<div class="row mt-3">
    <div class="col-sm mt-0 mb-0">
        {%% include figure.liquid loading="eager" path="%s" class="img-fluid rounded z-depth-1" zoomable=true %%}
    </div>
</div>`, imageURL)
	}

	return ""
}

// cleanText removes unwanted characters and fixes encoding issues
func cleanText(text string) string {
	if text == "" {
		return ""
	}

	// Replace non-breaking space (0xa0) with regular space
	text = strings.ReplaceAll(text, "\u00a0", " ")

	return text
}

func extractRichTextToMarkdown(blockContent map[string]any) string {
	richText, ok := blockContent["rich_text"].([]any)
	if !ok {
		return ""
	}

	var text string
	for _, rt := range richText {
		if rtMap, ok := rt.(map[string]any); ok {
			if plainText, ok := rtMap["plain_text"].(string); ok {
				// Apply formatting
				formattedText := applyRichTextFormatting(plainText, rtMap)
				text += formattedText
			}
		}
	}

	return text
}

func applyRichTextFormatting(text string, rtMap map[string]any) string {
	annotations, ok := rtMap["annotations"].(map[string]any)
	if !ok {
		return text
	}

	// Apply bold formatting
	if bold, ok := annotations["bold"].(bool); ok && bold {
		text = "**" + text + "**"
	}

	// Apply italic formatting
	if italic, ok := annotations["italic"].(bool); ok && italic {
		text = "*" + text + "*"
	}

	// Apply code formatting
	if code, ok := annotations["code"].(bool); ok && code {
		text = "`" + text + "`"
	}

	// Apply strikethrough formatting
	if strikethrough, ok := annotations["strikethrough"].(bool); ok && strikethrough {
		text = "~~" + text + "~~"
	}

	// Apply underline formatting (markdown doesn't have underline, use emphasis)
	if underline, ok := annotations["underline"].(bool); ok && underline {
		text = "*" + text + "*"
	}

	// Handle links
	if href, ok := rtMap["href"].(string); ok && href != "" {
		text = "[" + text + "](" + href + ")"
	}

	return text
}
