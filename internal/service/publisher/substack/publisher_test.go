package substack

import (
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
