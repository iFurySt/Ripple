package email

import (
	"context"
	"strings"
	"testing"

	"github.com/ifuryst/ripple/internal/service/publisher"
)

func TestParseRecipients(t *testing.T) {
	recipients := parseRecipients("a@example.com, b@example.com;c@example.com\n")
	if len(recipients) != 3 {
		t.Fatalf("expected 3 recipients, got %d", len(recipients))
	}
	if recipients[0] != "a@example.com" || recipients[1] != "b@example.com" || recipients[2] != "c@example.com" {
		t.Fatalf("unexpected recipients: %#v", recipients)
	}
}

func TestPublishDirectTransformsNotionBlocksBeforeSending(t *testing.T) {
	content := publisher.PublishContent{
		Title:   "Test",
		Content: `[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","plain_text":"hello","href":null,"annotations":{},"text":{"content":"hello"}}]}}]`,
	}

	transformed, err := (&Publisher{}).TransformContent(context.Background(), content)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(transformed.Content, `"object":"block"`) {
		t.Fatalf("expected transformed HTML, got raw blocks: %s", transformed.Content)
	}
	if !strings.Contains(transformed.Content, "hello") {
		t.Fatalf("expected transformed content to include text, got: %s", transformed.Content)
	}
}

func TestLooksLikeNotionBlocks(t *testing.T) {
	if !looksLikeNotionBlocks(`[{"type":"paragraph","paragraph":{}}]`) {
		t.Fatal("expected Notion blocks JSON to be detected")
	}
	if looksLikeNotionBlocks(`<p>already html</p>`) {
		t.Fatal("expected HTML not to be detected as Notion blocks")
	}
}

func TestConvertNotionBlocksToEmailHTMLIncludesTables(t *testing.T) {
	content := `[
		{"type":"table","table":{"table_width":2,"has_column_header":true,"has_row_header":false}},
		{"type":"table_row","table_row":{"cells":[[{"plain_text":"Name","annotations":{"bold":true}}],[{"plain_text":"Value","annotations":{}}]]}},
		{"type":"table_row","table_row":{"cells":[[{"plain_text":"A&B","annotations":{}}],[{"plain_text":"<ok>","annotations":{}}]]}}
	]`

	result, err := convertNotionBlocksToEmailHTML(content)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"<table", "<th", "<strong>Name</strong>", "A&amp;B", "&lt;ok&gt;"} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected table HTML to contain %q, got: %s", want, result)
		}
	}
}

func TestWrapEmailHTMLUsesDigestLayout(t *testing.T) {
	body := `<ul><li>Story</li></ul>`
	rendered := wrapEmailHTML("LeoTalk · Hacker News Daily · 2026.05.10", body)

	for _, want := range []string{
		"LeoTalk Digest",
		"Daily Hacker News highlights",
		"Sent by Ripple via Resend",
		body,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected wrapped email to contain %q, got: %s", want, rendered)
		}
	}
}
