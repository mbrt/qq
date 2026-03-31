package index

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"log/slog"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	index_api "github.com/blevesearch/bleve_index_api"

	"github.com/mbrt/qq/internal/document"
)

const searchMaxResults = 20

type Index struct {
	index bleve.Index
}

type ReconcileStats struct {
	Added     int
	Updated   int
	Removed   int
	Unchanged int
}

type SearchResult struct {
	Total int
	Took  time.Duration
	Hits  []SearchHit
}

type SearchHit struct {
	ID        string
	Score     float64
	Title     string
	Source    string
	Author    string
	URL       string
	Tags      []string
	Updated   time.Time
	Excerpt   string
	Fragments []string
}

// Open opens or creates a Bleve index at the given path.
func Open(path string) (*Index, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		// Try anyway; the directory might already exist.
	}
	idx, err := bleve.Open(path)
	if err != nil {
		// Index doesn't exist, create it.
		m := buildMapping()
		idx, err = bleve.New(path, m)
		if err != nil {
			return nil, fmt.Errorf("creating index at %q: %w", path, err)
		}
	}
	return &Index{index: idx}, nil
}

func (idx *Index) Close() error {
	return idx.index.Close()
}

// Reconcile updates the index to match the given documents.
// It adds new documents, removes stale ones, and updates changed ones.
func (idx *Index) Reconcile(_ context.Context, docs []document.Document) (ReconcileStats, error) {
	// Build a map of current documents by ID.
	docMap := make(map[string]document.Document, len(docs))
	for _, d := range docs {
		docMap[d.ID] = d
	}

	// Get all existing IDs from the index.
	existingIDs, err := idx.allIDs()
	if err != nil {
		return ReconcileStats{}, fmt.Errorf("listing index IDs: %w", err)
	}
	existingSet := make(map[string]bool, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = true
	}

	var stats ReconcileStats
	batch := idx.index.NewBatch()

	// Add or update documents.
	for id, doc := range docMap {
		if existingSet[id] {
			// Check if it needs updating by comparing timestamps.
			existingTime, err := idx.getUpdatedTime(id)
			if err == nil && !doc.Updated.After(existingTime) {
				stats.Unchanged++
				continue
			}
			stats.Updated++
		} else {
			stats.Added++
		}
		if err := batch.Index(id, docToIndex(doc)); err != nil {
			return stats, fmt.Errorf("indexing %q: %w", id, err)
		}
	}

	// Remove stale documents.
	for _, id := range existingIDs {
		if _, ok := docMap[id]; !ok {
			batch.Delete(id)
			stats.Removed++
		}
	}

	if batch.Size() > 0 {
		if err := idx.index.Batch(batch); err != nil {
			return stats, fmt.Errorf("applying batch: %w", err)
		}
	}
	return stats, nil
}

// Search executes a query string search and returns results.
func (idx *Index) Search(query string) (SearchResult, error) {
	q := bleve.NewQueryStringQuery(query)
	req := bleve.NewSearchRequest(q)
	req.Size = searchMaxResults
	req.Highlight = bleve.NewHighlightWithStyle("html")
	req.Highlight.Fields = []string{"contents"}
	req.Fields = []string{"title", "source", "author", "url", "tags", "updated", "excerpt"}

	res, err := idx.index.Search(req)
	if err != nil {
		return SearchResult{}, err
	}

	var hits []SearchHit
	for _, hit := range res.Hits {
		var fragments []string
		if c, ok := hit.Fragments["contents"]; ok {
			fragments = c
		}
		hits = append(hits, SearchHit{
			ID:        hit.ID,
			Score:     hit.Score,
			Title:     stringField(hit.Fields, "title"),
			Source:    stringField(hit.Fields, "source"),
			Author:    stringField(hit.Fields, "author"),
			URL:       stringField(hit.Fields, "url"),
			Tags:      arrayField(hit.Fields, "tags"),
			Updated:   timeField(hit.Fields, "updated"),
			Excerpt:   stringField(hit.Fields, "excerpt"),
			Fragments: fragments,
		})
	}

	return SearchResult{
		Total: int(res.Total),
		Took:  res.Took,
		Hits:  hits,
	}, nil
}

// Read retrieves a single document from the index by ID and returns its
// rendered HTML contents along with metadata.
func (idx *Index) Read(id string) (ReadResult, error) {
	doc, err := idx.index.Document(id)
	if err != nil {
		return ReadResult{}, err
	}
	if doc == nil {
		return ReadResult{}, fmt.Errorf("document %q not found", id)
	}
	result := ReadResult{}
	doc.VisitFields(func(f index_api.Field) {
		switch f.Name() {
		case "title":
			result.Title = fieldText(f)
		case "author":
			result.Author = fieldText(f)
		case "url":
			result.URL = fieldText(f)
		case "source":
			result.Source = fieldText(f)
		case "tags":
			if t := fieldText(f); t != "" {
				result.Tags = append(result.Tags, t)
			}
		case "excerpt":
			result.Excerpt = fieldText(f)
		case "contents":
			result.Contents = fieldText(f)
		case "updated":
			if df, ok := f.(index_api.DateTimeField); ok {
				if t, _, err := df.DateTime(); err == nil {
					result.Updated = t
				}
			}
		}
	})
	return result, nil
}

type ReadResult struct {
	Title        string
	Author       string
	URL          string
	Source       string
	Tags         []string
	Updated      time.Time
	Excerpt      string
	Contents     string
	HTMLContents template.HTML
}

func (idx *Index) allIDs() ([]string, error) {
	var ids []string
	const batchSize = 1000
	from := 0
	for {
		req := bleve.NewSearchRequest(bleve.NewMatchAllQuery())
		req.Size = batchSize
		req.From = from
		res, err := idx.index.Search(req)
		if err != nil {
			return nil, err
		}
		for _, hit := range res.Hits {
			ids = append(ids, hit.ID)
		}
		if len(res.Hits) < batchSize {
			break
		}
		from += batchSize
	}
	return ids, nil
}

func (idx *Index) getUpdatedTime(id string) (time.Time, error) {
	doc, err := idx.index.Document(id)
	if err != nil || doc == nil {
		return time.Time{}, fmt.Errorf("document %q not found", id)
	}
	var updated time.Time
	doc.VisitFields(func(f index_api.Field) {
		if f.Name() != "updated" {
			return
		}
		if df, ok := f.(index_api.DateTimeField); ok {
			if t, _, err := df.DateTime(); err == nil {
				updated = t
			}
		}
	})
	return updated, nil
}

type indexDoc struct {
	Title    string    `json:"title"`
	Author   string    `json:"author,omitempty"`
	URL      string    `json:"url,omitempty"`
	Source   string    `json:"source,omitempty"`
	Tags     []string  `json:"tags,omitempty"`
	Updated  time.Time `json:"updated"`
	Excerpt  string    `json:"excerpt,omitempty"`
	Contents string    `json:"contents"`
}

func docToIndex(d document.Document) indexDoc {
	return indexDoc{
		Title:    d.Title,
		Author:   d.Author,
		URL:      d.URL,
		Source:   d.Source,
		Tags:     d.Tags,
		Updated:  d.Updated,
		Excerpt:  d.Excerpt,
		Contents: d.Contents,
	}
}

func buildMapping() mapping.IndexMapping {
	textField := bleve.NewTextFieldMapping()
	textField.Store = true

	keywordField := bleve.NewKeywordFieldMapping()
	keywordField.Store = true

	dateField := bleve.NewDateTimeFieldMapping()
	dateField.Store = true

	storedOnlyText := bleve.NewTextFieldMapping()
	storedOnlyText.Store = true
	storedOnlyText.Index = false

	storedOnlyKeyword := bleve.NewKeywordFieldMapping()
	storedOnlyKeyword.Store = true
	storedOnlyKeyword.Index = false

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("title", textField)
	docMapping.AddFieldMappingsAt("contents", textField)
	docMapping.AddFieldMappingsAt("tags", keywordField)
	docMapping.AddFieldMappingsAt("source", keywordField)
	docMapping.AddFieldMappingsAt("author", textField)
	docMapping.AddFieldMappingsAt("updated", dateField)
	docMapping.AddFieldMappingsAt("url", storedOnlyKeyword)
	docMapping.AddFieldMappingsAt("excerpt", storedOnlyText)

	m := bleve.NewIndexMapping()
	m.DefaultMapping = docMapping
	return m
}

func fieldText(f index_api.Field) string {
	if tf, ok := f.(index_api.TextField); ok {
		return tf.Text()
	}
	return ""
}

func stringField(fs map[string]any, name string) string {
	f, ok := fs[name]
	if !ok {
		return ""
	}
	if s, ok := f.(string); ok {
		return s
	}
	return ""
}

func arrayField(fs map[string]any, name string) []string {
	f, ok := fs[name]
	if !ok {
		return nil
	}
	if s, ok := f.(string); ok {
		return []string{s}
	}
	ss, ok := f.([]any)
	if !ok {
		return nil
	}
	var res []string
	for _, s := range ss {
		if s, ok := s.(string); ok {
			res = append(res, s)
		}
	}
	return res
}

func timeField(fs map[string]any, name string) time.Time {
	s := stringField(fs, name)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	slog.Warn("Failed to parse time field", "field", name, "value", s)
	return time.Time{}
}

// StripHTMLTags removes HTML tags from a string (for CLI display of fragments).
func StripHTMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteRune(r)
		}
	}
	return b.String()
}
