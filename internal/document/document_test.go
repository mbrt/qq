package document

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParse_KBStyle(t *testing.T) {
	data := []byte(`---
title: A forty-year career.
author: Will Larson
url: https://lethain.com/forty-year-career/
source: instapaper
date: 2019-10-08T06:00:00-07:00
saved: 2026-01-11T07:47:12Z
tags:
    - manage
---

The Silicon Valley narrative centers on entrepreneurial protagonists.
`)
	mtime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	doc := Parse("articles/2026/w02/a-forty-year-career.md", "a-forty-year-career", data, mtime)

	assert.Equal(t, "articles/2026/w02/a-forty-year-career.md", doc.ID)
	assert.Equal(t, "A forty-year career.", doc.Title)
	assert.Equal(t, "Will Larson", doc.Author)
	assert.Equal(t, "https://lethain.com/forty-year-career/", doc.URL)
	assert.Equal(t, "instapaper", doc.Source)
	assert.Equal(t, []string{"manage"}, doc.Tags)
	assert.Equal(t, time.Date(2026, 1, 11, 7, 47, 12, 0, time.UTC), doc.Updated)
	assert.Contains(t, doc.Contents, "Silicon Valley")
	assert.NotEmpty(t, doc.Excerpt)
}

func TestParse_ObsidianStyle(t *testing.T) {
	data := []byte(`---
tags:
  - ai
updated: '2016-06-15'
---

Benchmarking is about running programs to test their performances.
`)
	mtime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	doc := Parse("languages/theory/Benchmarks.md", "Benchmarks", data, mtime)

	assert.Equal(t, "Benchmarks", doc.Title)
	assert.Equal(t, []string{"ai"}, doc.Tags)
	assert.Equal(t, time.Date(2016, 6, 15, 0, 0, 0, 0, time.UTC), doc.Updated)
	assert.Contains(t, doc.Contents, "Benchmarking")
}

func TestParse_NoFrontmatter(t *testing.T) {
	data := []byte(`Need to add a drift stopper. Alternatives:
- Duotone drift stopper
- Slingshot SUP Winger
`)
	mtime := time.Date(2025, 5, 10, 12, 0, 0, 0, time.UTC)
	doc := Parse("Keep/SUP wing.md", "SUP wing", data, mtime)

	assert.Equal(t, "SUP wing", doc.Title)
	assert.Empty(t, doc.Tags)
	assert.Equal(t, mtime, doc.Updated)
	assert.Contains(t, doc.Contents, "drift stopper")
}

func TestParse_EmptyFile(t *testing.T) {
	doc := Parse("empty.md", "empty", []byte{}, time.Now())
	assert.Equal(t, "empty", doc.Title)
	assert.Empty(t, doc.Contents)
	assert.Empty(t, doc.Excerpt)
}

func TestParse_FrontmatterOnly(t *testing.T) {
	data := []byte(`---
title: Stub Article
url: https://example.com/stub
saved: 2024-06-01T10:00:00Z
---
`)
	doc := Parse("stub.md", "stub", data, time.Now())
	assert.Equal(t, "Stub Article", doc.Title)
	assert.Empty(t, doc.Tags)
}

func TestParse_UpdatedPriority(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantYear int
	}{
		{
			name:     "saved wins over date and updated",
			data:     "---\nsaved: 2026-01-01T00:00:00Z\ndate: 2020-01-01T00:00:00Z\nupdated: '2015-01-01'\n---\n",
			wantYear: 2026,
		},
		{
			name:     "date wins when no saved",
			data:     "---\ndate: 2020-06-15T00:00:00Z\nupdated: '2015-01-01'\n---\n",
			wantYear: 2020,
		},
		{
			name:     "updated wins when no saved or date",
			data:     "---\nupdated: '2015-03-20'\n---\n",
			wantYear: 2015,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mtime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
			doc := Parse("test.md", "test", []byte(tt.data), mtime)
			assert.Equal(t, tt.wantYear, doc.Updated.Year())
		})
	}
}

func TestMakeExcerpt_Long(t *testing.T) {
	long := make([]byte, 500)
	for i := range long {
		long[i] = 'a'
	}
	excerpt := makeExcerpt(string(long))
	assert.Len(t, excerpt, maxExcerptLen)
}

func TestCollapseWhitespace(t *testing.T) {
	assert.Equal(t, "hello world", collapseWhitespace("hello\n\n  world"))
	assert.Equal(t, "a b c", collapseWhitespace("a  b\t\nc"))
}
