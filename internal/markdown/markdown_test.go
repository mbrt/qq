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
			html, err := ToHTML(tt.input)
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
