package wechat_official

import (
	"context"
	"fmt"
	"github.com/ifuryst/ripple/internal/service/publisher"
	"regexp"
	"strings"
)

// WeChatTransformer converts content to WeChat Official Account format
type WeChatTransformer struct{}

func NewWeChatTransformer() *WeChatTransformer {
	return &WeChatTransformer{}
}

func (t *WeChatTransformer) TransformContent(ctx context.Context, content publisher.PublishContent) (*publisher.PublishContent, error) {
	// Convert Notion blocks JSON directly to WeChat HTML
	wechatHTML, err := convertNotionBlocksToWeChatHTML(content.Content)
	if err != nil {
		return nil, fmt.Errorf("notion blocks to WeChat HTML conversion failed: %w", err)
	}

	// Extract links and add references
	wechatHTML, err = t.extractLinksAndAddReferences(wechatHTML)
	if err != nil {
		return nil, fmt.Errorf("link extraction failed: %w", err)
	}

	// Wrap in container
	wechatHTML = t.wrapInContainer(wechatHTML)

	result := content
	result.Content = wechatHTML
	return &result, nil
}

func (t *WeChatTransformer) wrapInContainer(content string) string {
	// Use WeChat reference base styling
	return content
}

// UpdateImageReferences updates image references with WeChat image URLs
func (t *WeChatTransformer) UpdateImageReferences(content string, resources []publisher.Resource) string {
	for _, resource := range resources {
		if resource.Type == publisher.ResourceTypeImage {
			wechatImageURL := resource.Metadata["wechat_image_url"]

			if wechatImageURL != "" && resource.URL != "" {
				// Replace original image URL with WeChat permanent image URL
				oldImg := fmt.Sprintf(`<img src="%s"`, resource.URL)
				newImg := fmt.Sprintf(`<img src="%s"`, wechatImageURL)
				content = strings.ReplaceAll(content, oldImg, newImg)

				// Also replace any other references to the original URL
				content = strings.ReplaceAll(content, resource.URL, wechatImageURL)
			}
		}
	}
	return content
}

// ExtractImages extracts image URLs from content for processing
func (t *WeChatTransformer) ExtractImages(content string) []string {
	var urls []string

	imageRegex := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)
	matches := imageRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			url := match[1]
			// Skip WeChat URLs (already processed) and empty URLs
			if url != "" && !strings.Contains(url, "mmbiz.qpic.cn") && !strings.Contains(url, "data-media-id") {
				urls = append(urls, url)
			}
		}
	}

	return urls
}

// LinkInfo represents link information for references
type LinkInfo struct {
	URL  string
	Text string
}

// extractLinksAndAddReferences extracts links from content and adds reference section
func (t *WeChatTransformer) extractLinksAndAddReferences(content string) (string, error) {
	// Extract all links from the content
	linkRegex := regexp.MustCompile(`<a\s+[^>]*href=["']([^"']+)["'][^>]*>([^<]+)</a>`)
	matches := linkRegex.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return content, nil
	}

	// Store unique links and their display text
	linkMap := make(map[string]LinkInfo)
	var linkOrder []string

	for _, match := range matches {
		if len(match) >= 3 {
			url := match[1]
			text := match[2]

			// Skip if URL is empty or already processed
			if url == "" {
				continue
			}

			// Use URL as key to avoid duplicates
			if _, exists := linkMap[url]; !exists {
				linkMap[url] = LinkInfo{URL: url, Text: text}
				linkOrder = append(linkOrder, url)
			}
		}
	}

	if len(linkOrder) == 0 {
		return content, nil
	}

	// Replace links with reference numbers
	modifiedContent := content
	for i, url := range linkOrder {
		refNum := i + 1
		linkInfo := linkMap[url]

		// Create reference link with superscript
		refLink := fmt.Sprintf(`<span style="text-align:left;color:#ff3502;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:16px">%s<sup>[%d]</sup></span>`,
			linkInfo.Text, refNum)

		// Replace the original link
		originalLink := fmt.Sprintf(`<a href="%s" style="color: #3498db; text-decoration: none; border-bottom: 1px dotted #3498db;">%s</a>`, url, linkInfo.Text)
		modifiedContent = strings.ReplaceAll(modifiedContent, originalLink, refLink)
	}

	// Add References section
	referencesHTML := t.generateReferencesSection(linkOrder, linkMap)
	modifiedContent += referencesHTML

	return modifiedContent, nil
}

// generateReferencesSection creates the References section HTML
func (t *WeChatTransformer) generateReferencesSection(linkOrder []string, linkMap map[string]LinkInfo) string {
	if len(linkOrder) == 0 {
		return ""
	}

	var references strings.Builder

	// Add References header
	references.WriteString(`<h3 style="text-align:left;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:120%;margin:40px 10px 20px 10px;font-weight:bold">References</h3>`)

	// Add each reference
	for i, url := range linkOrder {
		refNum := i + 1
		linkInfo := linkMap[url]

		refHTML := fmt.Sprintf(`<p style="text-align:left;color:#3f3f3f;line-height:1.5;font-family:Optima-Regular, Optima, PingFangSC-light, PingFangTC-light, 'PingFang SC', Cambria, Cochin, Georgia, Times, 'Times New Roman', serif;font-size:14px;margin:10px 10px"><code style="font-size: 90%%; opacity: 0.6;">[%d]</code> %s: <i>%s</i><br></p>`,
			refNum, linkInfo.Text, linkInfo.URL)

		references.WriteString(refHTML)
	}

	return references.String()
}
