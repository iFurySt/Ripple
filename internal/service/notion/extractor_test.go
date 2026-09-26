package notion

import "testing"

func TestContentTypeFieldNames(t *testing.T) {
	for _, name := range []string{"Content type", "Content Type"} {
		properties := map[string]any{
			"Platform": map[string]any{"type": "multi_select", "multi_select": []any{map[string]any{"name": "Substack"}}},
			name:       map[string]any{"type": "multi_select", "multi_select": []any{map[string]any{"name": "AI"}}},
		}
		got := (&Service{}).extractContentType(properties)
		if len(got) != 1 || got[0] != "AI" {
			t.Fatalf("%s: got %v", name, got)
		}
	}
	got := (&Service{}).extractContentType(map[string]any{
		"Platform": map[string]any{"type": "multi_select", "multi_select": []any{map[string]any{"name": "Substack"}}},
	})
	if len(got) != 0 {
		t.Fatalf("missing Content Type must not use Platform: %v", got)
	}
}
