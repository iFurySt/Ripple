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

func TestTransformPreservesTablesAsMarkdownCodeBlocks(t *testing.T) {
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
	if len(doc.Content) != 1 || doc.Content[0].Type != "code_block" {
		t.Fatalf("expected a markdown code block for table fallback, got: %#v", doc.Content)
	}
	if got := doc.Content[0].Content[0].Text; !strings.Contains(got, "| Name | Value |") || !strings.Contains(got, "| A | B |") {
		t.Fatalf("expected markdown table content, got: %s", got)
	}
}
