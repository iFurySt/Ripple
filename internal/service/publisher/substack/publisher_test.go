package substack

import (
	"context"
	"github.com/ifuryst/ripple/internal/models"
	"github.com/ifuryst/ripple/internal/service/publisher"
	"go.uber.org/zap"
	"reflect"
	"testing"
)

func TestNormalizeSubstackTagNames(t *testing.T) {
	tags := normalizeSubstackTagNames([]string{" AI ", "ai", "", "Browser Use"})

	expected := []string{"AI", "Browser Use"}
	if !reflect.DeepEqual(tags, expected) {
		t.Fatalf("expected %v, got %v", expected, tags)
	}
}

func TestFindSubstackTagMatchesNameCanonicalNameAndSlug(t *testing.T) {
	tags := []SubstackPublicationTag{
		{ID: "tag-1", Name: "Technology", CanonicalName: "Technology", Slug: "technology"},
		{ID: "tag-2", Name: "Browser Use", CanonicalName: "Browser Use", Slug: "browser-use"},
	}

	if tag := findSubstackTag(tags, "technology"); tag == nil || tag.ID != "tag-1" {
		t.Fatalf("expected to match tag 1 by name/canonical name, got %#v", tag)
	}

	if tag := findSubstackTag(tags, "browser-use"); tag == nil || tag.ID != "tag-2" {
		t.Fatalf("expected to match tag 2 by slug, got %#v", tag)
	}
}

func TestNotionContentTypeSuppliesSubstackTags(t *testing.T) {
	for _, types := range []models.StringArray{{"AI", "Research, Notes"}, {}} {
		page := &models.NotionPage{
			Content:     "[]",
			Tags:        models.StringArray{"Blog", "Substack"},
			Platforms:   models.StringArray{"Blog", "Substack"},
			ContentType: types,
		}
		content := publisher.FromNotionPage(page)
		p := NewSubstackPublisher(zap.NewNop())
		transformed, err := p.TransformContent(context.Background(), *content)
		if err != nil {
			t.Fatal(err)
		}
		if len(transformed.Tags) != len(types) {
			t.Fatalf("expected content types %v, got %v", types, transformed.Tags)
		}
		for i := range types {
			if transformed.Tags[i] != types[i] {
				t.Fatalf("expected content types %v, got %v", types, transformed.Tags)
			}
		}
	}
}
