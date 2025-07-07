package substack

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ifuryst/ripple/internal/service/publisher"
)

// SubstackTransformer transforms content for Substack publication
type SubstackTransformer struct {
	imageURLPattern *regexp.Regexp
}

// SubstackDocument represents Substack's document structure
type SubstackDocument struct {
	Type    string               `json:"type"`
	Content []SubstackNode       `json:"content"`
}

type SubstackNode struct {
	Type    string                 `json:"type"`
	Content []SubstackNode         `json:"content,omitempty"`
	Attrs   map[string]interface{} `json:"attrs,omitempty"`
	Marks   []SubstackMark         `json:"marks,omitempty"`
	Text    string                 `json:"text,omitempty"`
}

type SubstackMark struct {
	Type  string                 `json:"type"`
	Attrs map[string]interface{} `json:"attrs,omitempty"`
}

func NewSubstackTransformer() *SubstackTransformer {
	return &SubstackTransformer{
		imageURLPattern: regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`),
	}
}

func (t *SubstackTransformer) Transform(ctx context.Context, content string) (string, error) {
	// Convert Notion blocks to Substack format
	document, err := t.convertNotionBlocksToSubstack(content)
	if err != nil {
		return "", fmt.Errorf("failed to convert Notion blocks to Substack format: %w", err)
	}
	
	// Serialize to JSON string
	jsonBytes, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("failed to serialize Substack document: %w", err)
	}

	return string(jsonBytes), nil
}

func (t *SubstackTransformer) ExtractImages(content string) []string {
	var imageURLs []string
	
	// Try to parse as Notion blocks JSON first
	var blocks []map[string]any
	if err := json.Unmarshal([]byte(content), &blocks); err == nil {
		// This is Notion blocks JSON, extract images from blocks
		for _, block := range blocks {
			if blockType, ok := block["type"].(string); ok && blockType == "image" {
				if blockContent, ok := block["image"].(map[string]any); ok {
					imageURL := t.extractImageURLFromBlock(blockContent)
					if imageURL != "" {
						imageURLs = append(imageURLs, imageURL)
					}
				}
			}
		}
	} else {
		// Fallback to markdown pattern matching
		matches := t.imageURLPattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				imageURLs = append(imageURLs, match[2])
			}
		}
	}
	
	return imageURLs
}

func (t *SubstackTransformer) extractImageURLFromBlock(blockContent map[string]any) string {
	// Try to get from file object (for uploaded images)
	if fileObj, ok := blockContent["file"].(map[string]any); ok {
		if url, ok := fileObj["url"].(string); ok {
			return url
		}
	}

	// Try to get from external object (for external images)
	if externalObj, ok := blockContent["external"].(map[string]any); ok {
		if url, ok := externalObj["url"].(string); ok {
			return url
		}
	}

	return ""
}

func (t *SubstackTransformer) UpdateImageReferences(content string, resources []publisher.Resource) string {
	result := content
	
	for _, resource := range resources {
		if resource.Type == publisher.ResourceTypeImage && resource.Metadata["uploaded_url"] != "" {
			originalURL := resource.Metadata["original_url"]
			uploadedURL := resource.Metadata["uploaded_url"]
			
			// Update image references in the JSON content
			result = strings.ReplaceAll(result, originalURL, uploadedURL)
		}
	}
	
	return result
}

func (t *SubstackTransformer) convertNotionBlocksToSubstack(blocksJSON string) (SubstackDocument, error) {
	var blocks []map[string]any
	if err := json.Unmarshal([]byte(blocksJSON), &blocks); err != nil {
		return SubstackDocument{}, fmt.Errorf("failed to unmarshal Notion blocks: %w", err)
	}

	var nodes []SubstackNode
	var currentBulletList []SubstackNode
	var currentOrderedList []SubstackNode
	numberedListCounter := 0

	for i, block := range blocks {
		substackNode, skip, isNumberedList, isBulletList := t.convertBlockToSubstack(block, &numberedListCounter)
		if skip {
			continue
		}

		// Handle list grouping
		if isBulletList {
			currentBulletList = append(currentBulletList, substackNode)
			// Check if next block is also a bullet list item
			if i+1 < len(blocks) {
				nextBlockType, _ := blocks[i+1]["type"].(string)
				if nextBlockType != "bulleted_list_item" {
					// End of bullet list
					nodes = append(nodes, SubstackNode{
						Type:    "bullet_list",
						Content: currentBulletList,
					})
					currentBulletList = nil
				}
			} else {
				// Last block, end bullet list
				nodes = append(nodes, SubstackNode{
					Type:    "bullet_list",
					Content: currentBulletList,
				})
				currentBulletList = nil
			}
		} else if isNumberedList {
			currentOrderedList = append(currentOrderedList, substackNode)
			// Check if next block is also a numbered list item
			if i+1 < len(blocks) {
				nextBlockType, _ := blocks[i+1]["type"].(string)
				if nextBlockType != "numbered_list_item" {
					// End of ordered list
					nodes = append(nodes, SubstackNode{
						Type: "ordered_list",
						Attrs: map[string]interface{}{
							"start": 1,
							"order": 1,
						},
						Content: currentOrderedList,
					})
					currentOrderedList = nil
					numberedListCounter = 0
				}
			} else {
				// Last block, end ordered list
				nodes = append(nodes, SubstackNode{
					Type: "ordered_list",
					Attrs: map[string]interface{}{
						"start": 1,
						"order": 1,
					},
					Content: currentOrderedList,
				})
				currentOrderedList = nil
				numberedListCounter = 0
			}
		} else {
			// Reset counters if this is not a list item
			numberedListCounter = 0
			if substackNode.Type != "" {
				nodes = append(nodes, substackNode)
			}
		}
	}

	return SubstackDocument{
		Type:    "doc",
		Content: nodes,
	}, nil
}

func (t *SubstackTransformer) convertBlockToSubstack(block map[string]any, numberedListCounter *int) (substackNode SubstackNode, skip bool, isNumberedList bool, isBulletList bool) {
	blockType, ok := block["type"].(string)
	if !ok {
		return SubstackNode{}, true, false, false
	}

	blockContent, ok := block[blockType].(map[string]any)
	if !ok {
		return SubstackNode{}, true, false, false
	}

	switch blockType {
	case "paragraph":
		content := t.extractRichTextToSubstack(blockContent)
		if len(content) > 0 {
			return SubstackNode{
				Type:    "paragraph",
				Content: content,
			}, false, false, false
		}
		return SubstackNode{}, true, false, false

	case "heading_1":
		content := t.extractRichTextToSubstack(blockContent)
		if len(content) > 0 {
			return SubstackNode{
				Type: "heading",
				Attrs: map[string]interface{}{
					"level": 1,
				},
				Content: content,
			}, false, false, false
		}
		return SubstackNode{}, true, false, false

	case "heading_2":
		content := t.extractRichTextToSubstack(blockContent)
		if len(content) > 0 {
			return SubstackNode{
				Type: "heading",
				Attrs: map[string]interface{}{
					"level": 2,
				},
				Content: content,
			}, false, false, false
		}
		return SubstackNode{}, true, false, false

	case "heading_3":
		content := t.extractRichTextToSubstack(blockContent)
		if len(content) > 0 {
			return SubstackNode{
				Type: "heading",
				Attrs: map[string]interface{}{
					"level": 3,
				},
				Content: content,
			}, false, false, false
		}
		return SubstackNode{}, true, false, false

	case "bulleted_list_item":
		content := t.extractRichTextToSubstack(blockContent)
		if len(content) > 0 {
			return SubstackNode{
				Type: "list_item",
				Content: []SubstackNode{
					{
						Type:    "paragraph",
						Content: content,
					},
				},
			}, false, false, true
		}
		return SubstackNode{}, true, false, false

	case "numbered_list_item":
		content := t.extractRichTextToSubstack(blockContent)
		if len(content) > 0 {
			*numberedListCounter++
			return SubstackNode{
				Type: "list_item",
				Content: []SubstackNode{
					{
						Type:    "paragraph",
						Content: content,
					},
				},
			}, false, true, false
		}
		return SubstackNode{}, true, false, false

	case "quote":
		content := t.extractRichTextToSubstack(blockContent)
		if len(content) > 0 {
			return SubstackNode{
				Type: "blockquote",
				Content: []SubstackNode{
					{
						Type:    "paragraph",
						Content: content,
					},
				},
			}, false, false, false
		}
		return SubstackNode{}, true, false, false

	case "code":
		text := t.extractPlainTextFromRichText(blockContent)
		language := ""
		if lang, ok := blockContent["language"].(string); ok && lang != "" {
			language = lang
		}
		if text != "" {
			return SubstackNode{
				Type: "code_block",
				Attrs: map[string]interface{}{
					"language": language,
				},
				Content: []SubstackNode{
					{
						Type: "text",
						Text: text,
					},
				},
			}, false, false, false
		}
		return SubstackNode{}, true, false, false

	case "divider":
		return SubstackNode{
			Type: "horizontal_rule",
		}, false, false, false

	case "image":
		return t.convertImageBlockToSubstack(blockContent), false, false, false

	case "column_list", "column":
		// These are container blocks, their content comes from children
		return SubstackNode{}, true, false, false

	default:
		// For other block types, try to extract text as a paragraph
		content := t.extractRichTextToSubstack(blockContent)
		if len(content) > 0 {
			return SubstackNode{
				Type:    "paragraph",
				Content: content,
			}, false, false, false
		}
		return SubstackNode{}, true, false, false
	}
}

func (t *SubstackTransformer) extractRichTextToSubstack(blockContent map[string]any) []SubstackNode {
	richText, ok := blockContent["rich_text"].([]any)
	if !ok {
		return []SubstackNode{}
	}

	var nodes []SubstackNode
	for _, rt := range richText {
		if rtMap, ok := rt.(map[string]any); ok {
			if plainText, ok := rtMap["plain_text"].(string); ok {
				node := t.applySubstackFormatting(plainText, rtMap)
				if node.Type != "" {
					nodes = append(nodes, node)
				}
			}
		}
	}

	return nodes
}

func (t *SubstackTransformer) extractPlainTextFromRichText(blockContent map[string]any) string {
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

func (t *SubstackTransformer) applySubstackFormatting(text string, rtMap map[string]any) SubstackNode {
	node := SubstackNode{
		Type: "text",
		Text: text,
	}

	var marks []SubstackMark

	// Check for annotations
	if annotations, ok := rtMap["annotations"].(map[string]any); ok {
		// Apply bold formatting
		if bold, ok := annotations["bold"].(bool); ok && bold {
			marks = append(marks, SubstackMark{Type: "strong"})
		}

		// Apply italic formatting
		if italic, ok := annotations["italic"].(bool); ok && italic {
			marks = append(marks, SubstackMark{Type: "em"})
		}

		// Apply code formatting
		if code, ok := annotations["code"].(bool); ok && code {
			marks = append(marks, SubstackMark{Type: "code"})
		}

		// Apply strikethrough formatting
		if strikethrough, ok := annotations["strikethrough"].(bool); ok && strikethrough {
			marks = append(marks, SubstackMark{Type: "strikethrough"})
		}
	}

	// Handle links
	if href, ok := rtMap["href"].(string); ok && href != "" {
		marks = append(marks, SubstackMark{
			Type: "link",
			Attrs: map[string]interface{}{
				"href":   href,
				"target": "_blank",
				"rel":    "noopener noreferrer nofollow",
				"class":  nil,
			},
		})
	}

	if len(marks) > 0 {
		node.Marks = marks
	}

	return node
}

func (t *SubstackTransformer) convertImageBlockToSubstack(blockContent map[string]any) SubstackNode {
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
		return SubstackNode{
			Type: "captionedImage",
			Content: []SubstackNode{
				{
					Type: "image2",
					Attrs: map[string]interface{}{
						"src":                imageURL,
						"srcNoWatermark":     nil,
						"fullscreen":         nil,
						"imageSize":          nil,
						"height":             nil,
						"width":              nil,
						"resizeWidth":        nil,
						"bytes":              nil,
						"alt":                alt,
						"title":              nil,
						"type":               "image/png",
						"href":               nil,
						"belowTheFold":       false,
						"topImage":           false,
						"internalRedirect":   "",
						"isProcessing":       false,
						"align":              nil,
						"offset":             false,
					},
				},
			},
		}
	}

	return SubstackNode{}
}

