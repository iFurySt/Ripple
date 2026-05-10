package substack

import (
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
