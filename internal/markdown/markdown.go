// Package markdown converts markdown text to HTML and plain text.
package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

var (
	reLink   = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
	reMarkup = regexp.MustCompile("[#\\[\\]`*_~]")

	sanitizer = bluemonday.UGCPolicy()
)

// ToHTML converts markdown to sanitized HTML.
func ToHTML(md string) (string, error) {
	var buf bytes.Buffer
	gm := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	if err := gm.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return sanitizer.Sanitize(buf.String()), nil
}

// ToText strips markdown formatting and returns plain text.
func ToText(md string) string {
	noLinks := reLink.ReplaceAllString(md, "$1")
	noMarkup := reMarkup.ReplaceAllString(noLinks, "")
	return strings.TrimSpace(sanitizer.Sanitize(noMarkup))
}
