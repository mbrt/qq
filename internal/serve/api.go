// Package serve implements the qq web server and API handlers.
package serve

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mbrt/qq/internal/config"
	"github.com/mbrt/qq/internal/index"
	"github.com/mbrt/qq/internal/markdown"
)

type apiHandler struct {
	index *index.Index
	dirs  []config.Directory
}

func (a *apiHandler) Search(_ context.Context, query string) (index.SearchResult, error) {
	return a.index.Search(query)
}

func (a *apiHandler) Read(_ context.Context, id string) (index.ReadResult, error) {
	result, err := a.index.Read(id)
	if err != nil {
		return index.ReadResult{}, statusError{http.StatusNotFound, fmt.Errorf("document %q: %w", id, err)}
	}

	// If we have the raw markdown contents, try to read the file from disk
	// for the freshest version.
	if content := a.readFromDisk(id); content != "" {
		result.Contents = content
	}

	if result.Contents != "" {
		html, err := markdown.ToHTML(result.Contents)
		if err != nil {
			return result, nil
		}
		result.HTMLContents = template.HTML(html)
	}
	return result, nil
}

// readFromDisk tries to find and read the original file for the given document ID.
func (a *apiHandler) readFromDisk(id string) string {
	for _, dir := range a.dirs {
		path := filepath.Join(dir.Path, id)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Strip frontmatter.
		return stripFrontmatter(string(data))
	}
	return ""
}

func stripFrontmatter(s string) string {
	if !strings.HasPrefix(s, "---\n") {
		return s
	}
	rest := s[4:]
	_, after, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return s
	}
	return strings.TrimPrefix(after, "\n")
}
