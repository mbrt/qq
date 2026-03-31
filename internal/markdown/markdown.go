// Package markdown converts markdown text to HTML and plain text.
package markdown

import (
	"bytes"
	"path"
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
	reImgSrc = regexp.MustCompile(`(<img\s[^>]*?\bsrc=")([^"]+)(")`)

	sanitizer = bluemonday.UGCPolicy()
)

// ToHTML converts markdown to sanitized HTML. If filesPrefix and baseDir are
// non-empty, relative image src attributes are rewritten to
// {filesPrefix}/{baseDir}/{src}. For example, with filesPrefix="/files/0" and
// baseDir="2026/w14", a relative src "img.png" becomes "/files/0/2026/w14/img.png".
func ToHTML(md string, filesPrefix string, baseDir string) (string, error) {
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
	html := sanitizer.Sanitize(buf.String())
	if filesPrefix != "" {
		html = rewriteImageSrcs(html, filesPrefix, baseDir)
	}
	return html, nil
}

func rewriteImageSrcs(html, filesPrefix, baseDir string) string {
	return reImgSrc.ReplaceAllStringFunc(html, func(match string) string {
		parts := reImgSrc.FindStringSubmatch(match)
		src := parts[2]
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "/") {
			return match
		}
		rewritten := filesPrefix + "/" + path.Join(baseDir, src)
		return parts[1] + rewritten + parts[3]
	})
}

// ToText strips markdown formatting and returns plain text.
func ToText(md string) string {
	noLinks := reLink.ReplaceAllString(md, "$1")
	noMarkup := reMarkup.ReplaceAllString(noLinks, "")
	return strings.TrimSpace(sanitizer.Sanitize(noMarkup))
}
