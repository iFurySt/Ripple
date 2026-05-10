package email

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

func looksLikeNotionBlocks(content string) bool {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "[") {
		return false
	}

	var blocks []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &blocks); err != nil {
		return false
	}

	return len(blocks) > 0
}

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
				content = append(content, `<ul style="list-style:none;padding:0;margin:18px 0 0;">`)
				inBulletedList = true
			}
			if text := richTextHTML(blockContent(block)); text != "" {
				content = append(content, fmt.Sprintf(`<li style="margin:0 0 12px;padding:16px 18px;border:1px solid #e3e8ef;border-radius:12px;background:#ffffff;box-shadow:0 1px 2px rgba(15,23,42,0.04);line-height:1.62;color:#1f2937;overflow-wrap:anywhere;">%s</li>`, text))
			}
		case "numbered_list_item":
			if !inNumberedList {
				content = append(content, `<ol style="padding:0;margin:18px 0 0;list-style-position:inside;">`)
				inNumberedList = true
			}
			numberedListCounter++
			if text := richTextHTML(blockContent(block)); text != "" {
				content = append(content, fmt.Sprintf(`<li style="margin:0 0 12px;padding:16px 18px;border:1px solid #e3e8ef;border-radius:12px;background:#ffffff;box-shadow:0 1px 2px rgba(15,23,42,0.04);line-height:1.62;color:#1f2937;overflow-wrap:anywhere;">%s</li>`, text))
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
			return fmt.Sprintf(`<p style="line-height:1.72;margin:14px 0;color:#334155;">%s</p>`, text)
		}
	case "heading_1":
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<h1 style="font-size:28px;line-height:1.3;margin:28px 0 16px;color:#111827;">%s</h1>`, text)
		}
	case "heading_2":
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<h2 style="font-size:22px;line-height:1.35;margin:24px 0 14px;color:#111827;">%s</h2>`, text)
		}
	case "heading_3":
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<h3 style="font-size:18px;line-height:1.4;margin:20px 0 12px;color:#111827;">%s</h3>`, text)
		}
	case "quote":
		if text := richTextHTML(content); text != "" {
			return fmt.Sprintf(`<blockquote style="border-left:4px solid #2563eb;margin:16px 0;padding:10px 16px;color:#475569;background:#f8fafc;">%s</blockquote>`, text)
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
			return fmt.Sprintf(`<p style="line-height:1.72;margin:14px 0;color:#334155;">%s</p>`, text)
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
			formatted = fmt.Sprintf(`<a href="%s" style="color:#2563eb;text-decoration:none;border-bottom:1px solid #bfdbfe;overflow-wrap:anywhere;">%s</a>`, html.EscapeString(href), formatted)
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
	escapedTitle := html.EscapeString(title)

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>%s</title>
</head>
<body style="margin:0;background:#edf2f7;color:#1f2937;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;">
  <div style="display:none;max-height:0;overflow:hidden;color:#edf2f7;opacity:0;">A compact Hacker News digest from LeoTalk.</div>
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;background:#edf2f7;">
    <tr>
      <td align="center" style="padding:28px 12px;">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;max-width:760px;background:#f8fafc;border-radius:24px;overflow:hidden;border:1px solid #dbe4ee;">
          <tr>
            <td style="padding:32px 28px 30px;background:#111827;color:#ffffff;">
              <div style="font-size:12px;font-weight:700;letter-spacing:0.08em;text-transform:uppercase;color:#93c5fd;margin-bottom:12px;">LeoTalk Digest</div>
              <h1 style="font-size:34px;line-height:1.18;margin:0 0 12px;font-weight:800;color:#ffffff;">%s</h1>
              <p style="font-size:15px;line-height:1.7;margin:0;color:#cbd5e1;">Daily Hacker News highlights, filtered for engineers and product builders.</p>
            </td>
          </tr>
          <tr>
            <td style="padding:26px 28px 30px;background:#f8fafc;">
              %s
            </td>
          </tr>
          <tr>
            <td style="padding:18px 28px 26px;background:#f8fafc;color:#64748b;font-size:12px;line-height:1.6;border-top:1px solid #e2e8f0;">
              Sent by Ripple via Resend. You are receiving this because HN Daily is enabled for this mailbox.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, escapedTitle, escapedTitle, body)
}
