package substack

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ifuryst/ripple/internal/service/publisher"
)

func TestUpdateImageReferencesReplacesJSONEscapedURLs(t *testing.T) {
	transformer := NewSubstackTransformer()
	originalURL := "https://prod-files-secure.s3.us-west-2.amazonaws.com/image.png?X-Amz-Date=20260510&X-Amz-Signature=abc"
	uploadedURL := "https://substack-post-media.s3.amazonaws.com/public/images/image.png"
	content := `{"type":"doc","content":[{"type":"image2","attrs":{"src":"https://prod-files-secure.s3.us-west-2.amazonaws.com/image.png?X-Amz-Date=20260510\u0026X-Amz-Signature=abc"}}]}`

	result := transformer.UpdateImageReferences(content, []publisher.Resource{
		{
			Type: publisher.ResourceTypeImage,
			Metadata: map[string]string{
				"original_url": originalURL,
				"uploaded_url": uploadedURL,
			},
		},
	})

	if result == content {
		t.Fatal("expected image URL to be replaced")
	}
	if !strings.Contains(result, uploadedURL) {
		t.Fatalf("expected result to contain uploaded URL, got %s", result)
	}
	if strings.Contains(result, "prod-files-secure.s3.us-west-2.amazonaws.com") {
		t.Fatalf("expected original URL to be removed, got %s", result)
	}
}

func TestTransformPreservesTablesAsStructuredLists(t *testing.T) {
	transformer := NewSubstackTransformer()
	content := `[
		{"type":"table","table":{"table_width":2,"has_column_header":true,"has_row_header":false}},
		{"type":"table_row","table_row":{"cells":[[{"plain_text":"Name","annotations":{}}],[{"plain_text":"Value","annotations":{}}]]}},
		{"type":"table_row","table_row":{"cells":[[{"plain_text":"A","annotations":{}}],[{"plain_text":"B","annotations":{}}]]}}
	]`

	result, err := transformer.Transform(nil, content)
	if err != nil {
		t.Fatal(err)
	}

	var doc SubstackDocument
	if err := json.Unmarshal([]byte(result), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Content) != 1 || doc.Content[0].Type != "bullet_list" {
		t.Fatalf("expected a bullet list for table fallback, got: %#v", doc.Content)
	}
	item := doc.Content[0].Content[0]
	if item.Type != "list_item" || len(item.Content) != 1 {
		t.Fatalf("expected table row to become a list item, got: %#v", item)
	}
	paragraph := item.Content[0]
	if paragraph.Type != "paragraph" {
		t.Fatalf("expected list item content to be a paragraph, got: %#v", paragraph)
	}
	got := ""
	hasStrongHeader := false
	for _, node := range paragraph.Content {
		got += node.Text
		if node.Text == "Name" && len(node.Marks) > 0 && node.Marks[0].Type == "strong" {
			hasStrongHeader = true
		}
	}
	if !strings.Contains(got, "Name: A") || !strings.Contains(got, "Value: B") {
		t.Fatalf("expected labeled table row content, got: %s", got)
	}
	if !hasStrongHeader {
		t.Fatalf("expected header label to be bold, got: %#v", paragraph.Content)
	}
}
