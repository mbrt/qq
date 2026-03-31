package document

import (
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const maxExcerptLen = 200

type Document struct {
	ID       string
	Path     string
	Source   string
	Title    string
	Author   string
	URL      string
	Tags     []string
	Updated  time.Time
	Excerpt  string
	Contents string
}

type rawFrontmatter struct {
	Title   string     `yaml:"title"`
	Author  string     `yaml:"author"`
	URL     string     `yaml:"url"`
	Source  string     `yaml:"source"`
	Date    *time.Time `yaml:"date"`
	Saved   *time.Time `yaml:"saved"`
	Updated string     `yaml:"updated"`
	Tags    []string   `yaml:"tags"`
}

// Parse reads a markdown file's raw bytes and metadata, returning a Document.
// The id is the stable identifier (typically the relative path), filename is
// the base name without extension, and mtime is the file's modification time.
func Parse(id, filename string, data []byte, mtime time.Time) Document {
	body, fm := splitFrontmatter(data)

	doc := Document{
		ID:       id,
		Contents: body,
	}

	if fm.Title != "" {
		doc.Title = fm.Title
	} else {
		doc.Title = filename
	}

	doc.Author = fm.Author
	doc.URL = fm.URL
	doc.Source = fm.Source
	doc.Tags = fm.Tags

	switch {
	case fm.Saved != nil:
		doc.Updated = *fm.Saved
	case fm.Date != nil:
		doc.Updated = *fm.Date
	case fm.Updated != "":
		if t, err := time.Parse("2006-01-02", fm.Updated); err == nil {
			doc.Updated = t
		} else {
			doc.Updated = mtime
		}
	default:
		doc.Updated = mtime
	}

	doc.Excerpt = makeExcerpt(body)
	return doc
}

func splitFrontmatter(data []byte) (body string, fm rawFrontmatter) {
	s := string(data)
	if !strings.HasPrefix(s, "---\n") {
		return s, rawFrontmatter{}
	}
	rest := s[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		// Check for --- at end of file.
		if strings.HasSuffix(rest, "\n---") {
			end = len(rest) - 3
		} else {
			return s, rawFrontmatter{}
		}
	}
	fmData := rest[:end]
	body = strings.TrimPrefix(rest[end+4:], "\n")
	if strings.HasSuffix(rest, "\n---") && end == len(rest)-3 {
		body = ""
	}

	_ = yaml.Unmarshal([]byte(fmData), &fm)
	return body, fm
}

func makeExcerpt(body string) string {
	// Strip leading whitespace and take the first N characters.
	s := strings.TrimSpace(body)
	if utf8.RuneCountInString(s) <= maxExcerptLen {
		return collapseWhitespace(s)
	}
	runes := []rune(s)
	return collapseWhitespace(string(runes[:maxExcerptLen]))
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			r = ' '
		}
		if r == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
		} else {
			prevSpace = false
		}
		b.WriteRune(r)
	}
	return b.String()
}
