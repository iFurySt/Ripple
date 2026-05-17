package wechat_official

import (
	"strings"
	"testing"
)

func TestConvertNotionBlocksToWeChatHTMLIncludesTables(t *testing.T) {
	content := `[
		{"type":"table","table":{"table_width":2,"has_column_header":true,"has_row_header":false}},
		{"type":"table_row","table_row":{"cells":[[{"plain_text":"Name","annotations":{"bold":true}}],[{"plain_text":"Value","annotations":{}}]]}},
		{"type":"table_row","table_row":{"cells":[[{"plain_text":"A&B","annotations":{}}],[{"plain_text":"<ok>","annotations":{}}]]}}
	]`

	result, err := convertNotionBlocksToWeChatHTML(content)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"<table", "<th", "<strong", "A&amp;B", "&lt;ok&gt;"} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected table HTML to contain %q, got: %s", want, result)
		}
	}
}
