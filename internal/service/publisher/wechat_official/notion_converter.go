package wechat_official

import (
	"encoding/json"
	"fmt"
	"strings"
)

// convertNotionBlocksToWeChatHTML converts raw Notion blocks JSON to WeChat HTML format
func convertNotionBlocksToWeChatHTML(blocksJSON string) (string, error) {
	var blocks []map[string]any
	if err := json.Unmarshal([]byte(blocksJSON), &blocks); err != nil {
		return "", fmt.Errorf("failed to unmarshal blocks: %w", err)
	}

	// Convert blocks to WeChat HTML format
	var content []string
	numberedListCounter := 0

	for _, block := range blocks {
		html, skip, isNumberedList := convertBlockToWeChatHTMLWithCounter(block, &numberedListCounter)
		if skip {
			continue
		}

		// Reset counter if this is not a numbered list item
		if !isNumberedList {
			numberedListCounter = 0
		}

		if html != "" {
			content = append(content, html)
		}
	}

	result := strings.Join(content, "")
	
	// Clean up non-breaking spaces (0xa0) and replace with regular spaces
	result = cleanWeChatText(result)
	
	return result, nil
}

func convertBlockToWeChatHTMLWithCounter(block map[string]any, numberedListCounter *int) (content string, skip bool, isNumberedList bool) {
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
		text := extractRichTextToWeChatHTML(blockContent)
		if text != "" {
			content = fmt.Sprintf(`<p style="text-align:left;color:#3f3f3f;line-height:1.6;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;margin:10px 10px">%s</p>`, text)
		}
		return
	case "heading_1":
		text := extractRichTextToWeChatHTML(blockContent)
		if text != "" {
			content = fmt.Sprintf(`<h2 style="text-align:center;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:140%%;margin:80px 10px 40px 10px;font-weight:normal">%s</h2>`, text)
		}
		return
	case "heading_2":
		text := extractRichTextToWeChatHTML(blockContent)
		if text != "" {
			content = fmt.Sprintf(`<h2 style="text-align:center;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:140%%;margin:80px 10px 40px 10px;font-weight:normal">%s</h2>`, text)
		}
		return
	case "heading_3":
		text := extractRichTextToWeChatHTML(blockContent)
		if text != "" {
			content = fmt.Sprintf(`<h3 style="text-align:left;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:120%%;margin:40px 10px 20px 10px;font-weight:bold">%s</h3>`, text)
		}
		return
	case "bulleted_list_item":
		text := extractRichTextToWeChatHTML(blockContent)
		if text != "" {
			content = fmt.Sprintf(`<p style="text-align:left;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;margin:20px 10px;margin-left:0;padding-left:20px;list-style:circle"><span style="text-align:left;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;text-indent:-20px;display:block;margin:10px 10px"><span style="margin-right: 10px;">•</span>%s</span></p>`, text)
		}
		return
	case "numbered_list_item":
		text := extractRichTextToWeChatHTML(blockContent)
		if text != "" {
			*numberedListCounter++
			content = fmt.Sprintf(`<p style="text-align:left;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;margin:20px 10px;margin-left:0;padding-left:20px"><span style="text-align:left;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;text-indent:-20px;display:block;margin:10px 10px"><span style="margin-right: 10px;">%d.</span>%s</span></p>`, *numberedListCounter, text)
			isNumberedList = true
		}
		return
	case "quote":
		text := extractRichTextToWeChatHTML(blockContent)
		if text != "" {
			quoteParagraph := fmt.Sprintf(`<p style="text-align:left;color:#3f3f3f;line-height:1.6;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;margin:10px 10px">%s</p>`, text)
			content = fmt.Sprintf(`<blockquote style="text-align:left;color:rgb(91, 91, 91);line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;margin:20px 10px;padding:1px 0 1px 10px;background:rgba(158, 158, 158, 0.1);border-left:3px solid rgb(158,158,158)">%s</blockquote>`, quoteParagraph)
		}
		return
	case "code":
		text := extractPlainTextFromRichText(blockContent)
		language := "bash" // default language
		if lang, ok := blockContent["language"].(string); ok && lang != "" {
			language = lang
		}
		if text != "" {
			lines := strings.Split(text, "\n")
			lineNumbers := ""
			for range lines {
				lineNumbers += "<li></li>"
			}
			
			codeLines := ""
			for _, line := range lines {
				if line == "" {
					line = " " // prevent empty lines from collapsing
				}
				codeLines += fmt.Sprintf(`<code><span class="code-snippet_outer">%s</span></code>`, escapeHTML(line))
			}
			
			content = fmt.Sprintf(`<section class="code-snippet__fix code-snippet__js"><ul class="code-snippet__line-index code-snippet__js">%s</ul><pre class="code-snippet__js" data-lang="%s">%s</pre></section>`, lineNumbers, language, codeLines)
		}
		return
	case "divider":
		content = `<hr style="margin: 40px 10px; border: none; border-top: 1px solid #ddd;">`
		return
	case "image":
		content = convertImageBlockToWeChatHTML(blockContent)
		return
	case "column_list", "column":
		// These are container blocks, their content comes from children
		content = ""
		return
	default:
		// For other block types, just extract the text as a paragraph
		text := extractRichTextToWeChatHTML(blockContent)
		if text != "" {
			content = fmt.Sprintf(`<p style="text-align:left;color:#3f3f3f;line-height:1.6;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;margin:10px 10px">%s</p>`, text)
		}
		return
	}

	return content, false, false
}

func convertImageBlockToWeChatHTML(blockContent map[string]any) string {
	// Extract image URL from different possible sources
	var imageURL string
	var alt string

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

	// Try to get caption for alt text
	if caption, ok := blockContent["caption"].([]any); ok && len(caption) > 0 {
		if captionMap, ok := caption[0].(map[string]any); ok {
			if plainText, ok := captionMap["plain_text"].(string); ok {
				alt = plainText
			}
		}
	}

	if imageURL != "" {
		return fmt.Sprintf(`<p style="text-align:left;color:#3f3f3f;line-height:1.6;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;margin:10px 10px"><img style="text-align:left;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px;margin:20px auto;border-radius:4px;display:block;width:100%%" src="%s" title="null" alt="%s"></p>`, imageURL, alt)
	}

	return ""
}

func extractRichTextToWeChatHTML(blockContent map[string]any) string {
	richText, ok := blockContent["rich_text"].([]any)
	if !ok {
		return ""
	}

	var text string
	for _, rt := range richText {
		if rtMap, ok := rt.(map[string]any); ok {
			if plainText, ok := rtMap["plain_text"].(string); ok {
				// Apply formatting and convert to HTML
				formattedText := applyWeChatHTMLFormatting(plainText, rtMap)
				text += formattedText
			}
		}
	}

	return text
}

func extractPlainTextFromRichText(blockContent map[string]any) string {
	richText, ok := blockContent["rich_text"].([]any)
	if !ok {
		return ""
	}

	var text string
	for _, rt := range richText {
		if rtMap, ok := rt.(map[string]any); ok {
			if plainText, ok := rtMap["plain_text"].(string); ok {
				text += plainText
			}
		}
	}

	return text
}

func applyWeChatHTMLFormatting(text string, rtMap map[string]any) string {
	annotations, ok := rtMap["annotations"].(map[string]any)
	if !ok {
		// Handle links without annotations - keep the original format for now, references will be processed later
		if href, ok := rtMap["href"].(string); ok && href != "" {
			return fmt.Sprintf(`<a href="%s" style="color: #3498db; text-decoration: none; border-bottom: 1px dotted #3498db;">%s</a>`, href, escapeHTML(text))
		}
		return escapeHTML(text)
	}

	// Escape HTML first
	text = escapeHTML(text)

	// Apply bold formatting
	if bold, ok := annotations["bold"].(bool); ok && bold {
		text = fmt.Sprintf(`<strong style="text-align:left;color:#ff3502;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px">%s</strong>`, text)
	}

	// Apply italic formatting
	if italic, ok := annotations["italic"].(bool); ok && italic {
		text = fmt.Sprintf(`<em style="color: #3498db; font-style: italic;">%s</em>`, text)
	}

	// Apply code formatting
	if code, ok := annotations["code"].(bool); ok && code {
		text = fmt.Sprintf(`<code style="text-align:left;color:#ff3502;line-height:1.5;font-family:Operator Mono, Consolas, Monaco, Menlo, monospace;font-size:90%%;background:#f8f5ec;padding:3px 5px;border-radius:2px">%s</code>`, text)
	}

	// Apply strikethrough formatting
	if strikethrough, ok := annotations["strikethrough"].(bool); ok && strikethrough {
		text = fmt.Sprintf(`<s>%s</s>`, text)
	}

	// Apply underline formatting
	if underline, ok := annotations["underline"].(bool); ok && underline {
		text = fmt.Sprintf(`<u>%s</u>`, text)
	}

	// Handle links - keep the original format for now, references will be processed later
	if href, ok := rtMap["href"].(string); ok && href != "" {
		text = fmt.Sprintf(`<a href="%s" style="color: #3498db; text-decoration: none; border-bottom: 1px dotted #3498db;">%s</a>`, href, text)
	}

	return text
}

func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	return text
}

// cleanWeChatText removes unwanted characters and fixes encoding issues
func cleanWeChatText(text string) string {
	if text == "" {
		return ""
	}

	// Replace non-breaking space (0xa0) with regular space
	text = strings.ReplaceAll(text, "\u00a0", " ")

	return text
}