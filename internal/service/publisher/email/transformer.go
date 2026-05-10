package email

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

func convertNotionBlocksToEmailHTML(blocksJSON string) (string, error) {
	var blocks []map[string]any
	if err := json.Unmarshal([]byte(blocksJSON), &blocks); err != nil {
		if strings.TrimSpace(blocksJSON) == "" {
			return "", nil
		}
		return fmt.Sprintf("<p>%s</p>", html.EscapeString(blocksJSON)), nil
	}

	var content []string
	numberedListCounter := 0
	inBulletedList := false
	inNumberedList := false

	closeLists := func() {
		if inBulletedList {
			content = append(content, "</ul>")
			inBulletedList = false
		}
		if inNumberedList {
			content = append(content, "</ol>")
			inNumberedList = false
			numberedListCounter = 0
		}
	}

	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		if blockType != "bulleted_list_item" && blockType != "numbered_list_item" {
			closeLists()
		}

		switch blockType {
		case "bulleted_list_item":
			if !inBulletedList {
				content = append(content, `<ul style="padding-left:24px;margin:12px 0;">`)
				inBulletedList = true
			}
			if text := richTextHTML(blockContent(block)); text != "" {
				content = append(content, fmt.Sprintf(`<li style="margin:6px 0;line-height:1.6;">%s</li>`, text))
			}
		case "numbered_list_item":
			if !inNumberedList {
				content = append(content, `<ol style="padding-left:24px;margin:12px 0;">`)
				inNumberedList = true
			}
			numberedListCounter++
			if text := richTextHTML(blockContent(block)); text != "" {
				content = append(content, fmt.Sprintf(`<li style="margin:6px 0;line-height:1.6;">%s</li>`, text))
			}
		default:
			if blockHTML := blockToEmailHTML(block); blockHTML != "" {
				content = append(content, blockHTML)
			}
		}
	}
	closeLists()

	return strings.Join(content, "\n"), nil
}

func blockToEmailHTML(block map[string]any) string {
	blockType, _ := block["type"].(string)
	content := blockContent(block)

	switch blockType {
	case "paragraph":
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<p style="line-height:1.7;margin:12px 0;">%s</p>`, text)
		}
	case "heading_1":
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<h1 style="font-size:28px;line-height:1.3;margin:28px 0 16px;">%s</h1>`, text)
		}
	case "heading_2":
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<h2 style="font-size:22px;line-height:1.35;margin:24px 0 14px;">%s</h2>`, text)
		}
	case "heading_3":
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<h3 style="font-size:18px;line-height:1.4;margin:20px 0 12px;">%s</h3>`, text)
		}
	case "quote":
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<blockquote style="border-left:4px solid #d0d7de;margin:16px 0;padding:8px 14px;color:#57606a;background:#f6f8fa;">%s</blockquote>`, text)
		}
	case "code":
		if text := plainText(content); text != "" {
			return fmt.Sprintf(`<pre style="background:#f6f8fa;border:1px solid #d0d7de;border-radius:6px;padding:14px;overflow:auto;"><code>%s</code></pre>`, html.EscapeString(text))
		}
	case "divider":
		return `<hr style="border:none;border-top:1px solid #d0d7de;margin:28px 0;">`
	case "image":
		return imageHTML(content)
	default:
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<p style="line-height:1.7;margin:12px 0;">%s</p>`, text)
		}
	}

	return ""
}

func blockContent(block map[string]any) map[string]any {
	blockType, _ := block["type"].(string)
	if content, ok := block[blockType].(map[string]any); ok {
		return content
	}
	return map[string]any{}
}

func richTextHTML(content map[string]any) string {
	richText, ok := content["rich_text"].([]any)
	if !ok {
		return ""
	}

	var out strings.Builder
	for _, item := range richText {
		rt, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text, _ := rt["plain_text"].(string)
		if text == "" {
			continue
		}
		formatted := html.EscapeString(text)
		if annotations, ok := rt["annotations"].(map[string]any); ok {
			if v, _ := annotations["code"].(bool); v {
				formatted = fmt.Sprintf("<code>%s</code>", formatted)
			}
			if v, _ := annotations["bold"].(bool); v {
				formatted = fmt.Sprintf("<strong>%s</strong>", formatted)
			}
			if v, _ := annotations["italic"].(bool); v {
				formatted = fmt.Sprintf("<em>%s</em>", formatted)
			}
			if v, _ := annotations["strikethrough"].(bool); v {
				formatted = fmt.Sprintf("<s>%s</s>", formatted)
			}
			if v, _ := annotations["underline"].(bool); v {
				formatted = fmt.Sprintf("<u>%s</u>", formatted)
			}
		}
		if href, _ := rt["href"].(string); href != "" {
			formatted = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(href), formatted)
		}
		out.WriteString(formatted)
	}

	return out.String()
}

func plainText(content map[string]any) string {
	richText, ok := content["rich_text"].([]any)
	if !ok {
		return ""
	}

	var out strings.Builder
	for _, item := range richText {
		if rt, ok := item.(map[string]any); ok {
			if text, ok := rt["plain_text"].(string); ok {
				out.WriteString(text)
			}
		}
	}
	return out.String()
}

func imageHTML(content map[string]any) string {
	imageURL := ""
	if fileObj, ok := content["file"].(map[string]any); ok {
		imageURL, _ = fileObj["url"].(string)
	}
	if imageURL == "" {
		if externalObj, ok := content["external"].(map[string]any); ok {
			imageURL, _ = externalObj["url"].(string)
		}
	}
	if imageURL == "" {
		return ""
	}

	alt := ""
	if caption, ok := content["caption"].([]any); ok && len(caption) > 0 {
		if captionMap, ok := caption[0].(map[string]any); ok {
			alt, _ = captionMap["plain_text"].(string)
		}
	}

	return fmt.Sprintf(`<p style="margin:18px 0;"><img src="%s" alt="%s" style="max-width:100%%;height:auto;border-radius:6px;"></p>`, html.EscapeString(imageURL), html.EscapeString(alt))
}

func wrapEmailHTML(title string, body string) string {
	return fmt.Sprintf(`<!doctype html>
<html>
<body style="margin:0;background:#f6f8fa;color:#24292f;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;">
  <main style="max-width:760px;margin:0 auto;padding:32px 20px;background:#ffffff;">
    <h1 style="font-size:30px;line-height:1.25;margin:0 0 24px;">%s</h1>
    %s
  </main>
</body>
</html>`, html.EscapeString(title), body)
}
