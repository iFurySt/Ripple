package al_folio

import (
	"strings"
	"testing"
)

func TestConvertNotionBlocksToMarkdownIncludesTables(t *testing.T) {
	content := `[
		{"type":"paragraph","paragraph":{"rich_text":[{"plain_text":"before","annotations":{}}]}},
		{"type":"table","table":{"table_width":2,"has_column_header":true,"has_row_header":false}},
		{"type":"table_row","table_row":{"cells":[[{"plain_text":"Name","annotations":{"bold":true}}],[{"plain_text":"Value","annotations":{}}]]}},
		{"type":"table_row","table_row":{"cells":[[{"plain_text":"A|B","annotations":{}}],[{"plain_text":"Line\nBreak","annotations":{}}]]}},
		{"type":"paragraph","paragraph":{"rich_text":[{"plain_text":"after","annotations":{}}]}}
	]`

	result, err := convertNotionBlocksToMarkdown(content)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"before",
		"| **Name** | Value |",
		"| --- | --- |",
		`| A\|B | Line<br>Break |`,
		"after",
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, result)
		}
	}
}

func TestConvertNotionBlocksToMarkdownNormalizesPlainTextCodeLanguage(t *testing.T) {
	content := `[
		{"type":"code","code":{"language":"plain text","rich_text":[{"plain_text":"hello\nworld","annotations":{}}]}}
	]`

	result, err := convertNotionBlocksToMarkdown(content)
	if err != nil {
		t.Fatal(err)
	}

	want := "```plaintext\nhello\nworld\n```"
	if result != want {
		t.Fatalf("expected %q, got:\n%s", want, result)
	}
}
