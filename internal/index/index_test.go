package index

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mbrt/qq/internal/document"
)

func TestOpenAndClose(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	idx, err := Open(dir)
	require.NoError(t, err)
	require.NoError(t, idx.Close())
}

func TestReconcile_AddNew(t *testing.T) {
	idx := openTestIndex(t)
	docs := []document.Document{
		{ID: "a.md", Title: "Alpha", Contents: "alpha content", Updated: time.Now()},
		{ID: "b.md", Title: "Beta", Contents: "beta content", Updated: time.Now()},
	}
	stats, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Added)
	assert.Equal(t, 0, stats.Updated)
	assert.Equal(t, 0, stats.Removed)
}

func TestReconcile_RemoveStale(t *testing.T) {
	idx := openTestIndex(t)
	docs := []document.Document{
		{ID: "a.md", Title: "Alpha", Contents: "alpha", Updated: time.Now()},
		{ID: "b.md", Title: "Beta", Contents: "beta", Updated: time.Now()},
	}
	_, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)

	// Remove b.md.
	stats, err := idx.Reconcile(context.Background(), docs[:1])
	require.NoError(t, err)
	assert.Equal(t, 0, stats.Added)
	assert.Equal(t, 1, stats.Removed)
	assert.Equal(t, 1, stats.Unchanged)
}

func TestReconcile_UpdateChanged(t *testing.T) {
	idx := openTestIndex(t)
	now := time.Now()
	docs := []document.Document{
		{ID: "a.md", Title: "Alpha", Contents: "alpha", Updated: now},
	}
	_, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)

	// Update with newer timestamp.
	docs[0].Updated = now.Add(time.Hour)
	docs[0].Contents = "alpha updated"
	stats, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)
	assert.Equal(t, 0, stats.Added)
	assert.Equal(t, 1, stats.Updated)
}

func TestReconcile_Unchanged(t *testing.T) {
	idx := openTestIndex(t)
	docs := []document.Document{
		{ID: "a.md", Title: "Alpha", Contents: "alpha", Updated: time.Now()},
	}
	_, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)

	stats, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)
	assert.Equal(t, 0, stats.Added)
	assert.Equal(t, 0, stats.Updated)
	assert.Equal(t, 0, stats.Removed)
	assert.Equal(t, 1, stats.Unchanged)
}

func TestSearch(t *testing.T) {
	idx := openTestIndex(t)
	docs := []document.Document{
		{ID: "a.md", Title: "Concurrency in Go", Contents: "goroutines and channels", Tags: []string{"go"}, Updated: time.Now()},
		{ID: "b.md", Title: "Python Basics", Contents: "lists and dictionaries", Tags: []string{"python"}, Updated: time.Now()},
	}
	_, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)

	res, err := idx.Search("goroutines")
	require.NoError(t, err)
	assert.Equal(t, 1, res.Total)
	assert.Equal(t, "Concurrency in Go", res.Hits[0].Title)
}

func TestSearch_TitleBoost(t *testing.T) {
	idx := openTestIndex(t)
	docs := []document.Document{
		{ID: "a.md", Title: "Kubernetes Guide", Contents: "containers and orchestration", Updated: time.Now()},
		{ID: "b.md", Title: "Container Basics", Contents: "kubernetes deployment patterns", Updated: time.Now()},
	}
	_, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)

	res, err := idx.Search("kubernetes")
	require.NoError(t, err)
	require.Equal(t, 2, res.Total)
	assert.Equal(t, "Kubernetes Guide", res.Hits[0].Title, "title match should rank higher")
}

func TestSearch_TagBoost(t *testing.T) {
	idx := openTestIndex(t)
	docs := []document.Document{
		{ID: "a.md", Title: "Article One", Contents: "general content", Tags: []string{"kubernetes"}, Updated: time.Now()},
		{ID: "b.md", Title: "Article Two", Contents: "kubernetes deployment patterns", Updated: time.Now()},
	}
	_, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)

	res, err := idx.Search("kubernetes")
	require.NoError(t, err)
	require.Equal(t, 2, res.Total)
	assert.Equal(t, "Article One", res.Hits[0].Title, "tag match should rank higher than body match")
}

func TestSearch_TagAlias(t *testing.T) {
	idx := openTestIndex(t)
	docs := []document.Document{
		{ID: "a.md", Title: "Go Article", Contents: "goroutines", Tags: []string{"work"}, Updated: time.Now()},
		{ID: "b.md", Title: "Python Article", Contents: "lists", Tags: []string{"personal"}, Updated: time.Now()},
	}
	_, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)

	tests := []struct {
		query string
		want  string
	}{
		{"tags:work", "Go Article"},
		{"tag:work", "Go Article"},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			res, err := idx.Search(tt.query)
			require.NoError(t, err)
			require.Equal(t, 1, res.Total)
			assert.Equal(t, tt.want, res.Hits[0].Title)
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"tag:work", "tags:work"},
		{"tags:work", "tags:work"},
		{"Tag:work", "tags:work"},
		{"TAG:work", "tags:work"},
		{"tag:work goroutines", "tags:work goroutines"},
		{"goroutines", "goroutines"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeQuery(tt.input))
		})
	}
}

func TestRead(t *testing.T) {
	idx := openTestIndex(t)
	docs := []document.Document{
		{
			ID:       "a.md",
			Title:    "Test Article",
			Author:   "Author",
			URL:      "https://example.com",
			Source:   "instapaper",
			Tags:     []string{"test"},
			Contents: "Some content here",
			Updated:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	_, err := idx.Reconcile(context.Background(), docs)
	require.NoError(t, err)

	result, err := idx.Read("a.md")
	require.NoError(t, err)
	assert.Equal(t, "Test Article", result.Title)
	assert.Equal(t, "Author", result.Author)
	assert.Equal(t, "https://example.com", result.URL)
	assert.Equal(t, "instapaper", result.Source)
	assert.Equal(t, []string{"test"}, result.Tags)
	assert.Contains(t, result.Contents, "Some content")
}

func TestRead_NotFound(t *testing.T) {
	idx := openTestIndex(t)
	_, err := idx.Read("nonexistent.md")
	assert.Error(t, err)
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello <mark>world</mark>", "hello world"},
		{"no tags", "no tags"},
		{"<b>bold</b> and <i>italic</i>", "bold and italic"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, StripHTMLTags(tt.input))
		})
	}
}

func openTestIndex(t *testing.T) *Index {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "idx")
	idx, err := Open(dir)
	require.NoError(t, err)
	t.Cleanup(func() { idx.Close() })
	return idx
}
