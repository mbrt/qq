package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "heading",
			input:    "# Hello",
			contains: "<h1",
		},
		{
			name:     "paragraph",
			input:    "Hello world.",
			contains: "<p>Hello world.</p>",
		},
		{
			name:     "bold",
			input:    "**bold**",
			contains: "<strong>bold</strong>",
		},
		{
			name:     "link",
			input:    "[example](https://example.com)",
			contains: `<a href="https://example.com"`,
		},
		{
			name:     "code block",
			input:    "```go\nfmt.Println(\"hi\")\n```",
			contains: "<code",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := ToHTML(tt.input, "", "")
			require.NoError(t, err)
			assert.Contains(t, html, tt.contains)
		})
	}
}

func TestToHTMLImageRewrite(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		filesPrefix string
		baseDir     string
		contains    string
	}{
		{
			name:        "relative image rewritten with dir index",
			input:       "![photo](images/photo.png)",
			filesPrefix: "/files/0",
			baseDir:     "2026/w14",
			contains:    `src="/files/0/2026/w14/images/photo.png"`,
		},
		{
			name:        "second directory index",
			input:       "![photo](photo.png)",
			filesPrefix: "/files/1",
			baseDir:     "notes",
			contains:    `src="/files/1/notes/photo.png"`,
		},
		{
			name:        "absolute image unchanged",
			input:       "![photo](/abs/photo.png)",
			filesPrefix: "/files/0",
			baseDir:     "2026/w14",
			contains:    `src="/abs/photo.png"`,
		},
		{
			name:        "https image unchanged",
			input:       "![photo](https://example.com/photo.png)",
			filesPrefix: "/files/0",
			baseDir:     "2026/w14",
			contains:    `src="https://example.com/photo.png"`,
		},
		{
			name:        "http image unchanged",
			input:       "![photo](http://example.com/photo.png)",
			filesPrefix: "/files/0",
			baseDir:     "2026/w14",
			contains:    `src="http://example.com/photo.png"`,
		},
		{
			name:        "no filesPrefix leaves image unchanged",
			input:       "![photo](images/photo.png)",
			filesPrefix: "",
			baseDir:     "2026/w14",
			contains:    `src="images/photo.png"`,
		},
		{
			name:        "root-level doc with dir index",
			input:       "![photo](photo.png)",
			filesPrefix: "/files/0",
			baseDir:     ".",
			contains:    `src="/files/0/photo.png"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := ToHTML(tt.input, tt.filesPrefix, tt.baseDir)
			require.NoError(t, err)
			assert.Contains(t, html, tt.contains)
		})
	}
}

func TestToText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strip link",
			input: "[example](https://example.com)",
			want:  "example",
		},
		{
			name:  "strip heading",
			input: "# Hello World",
			want:  "Hello World",
		},
		{
			name:  "strip bold",
			input: "**bold** text",
			want:  "bold text",
		},
		{
			name:  "plain text unchanged",
			input: "just text",
			want:  "just text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ToText(tt.input))
		})
	}
}
