package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mbrt/qq/internal/config"
)

func TestScan_BasicFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hello.md", "---\ntitle: Hello\n---\n\nHello world.")
	writeFile(t, dir, "sub/nested.md", "---\ntags:\n  - test\nupdated: '2020-01-01'\n---\n\nNested content.")
	writeFile(t, dir, "not-markdown.txt", "ignored")

	docs, err := Scan([]config.Directory{{Path: dir}})
	require.NoError(t, err)
	assert.Len(t, docs, 2)

	byID := map[string]struct{}{}
	for _, d := range docs {
		byID[d.ID] = struct{}{}
		assert.NotEmpty(t, d.Path)
		assert.NotEmpty(t, d.Title)
	}
	assert.Contains(t, byID, "hello.md")
	assert.Contains(t, byID, filepath.Join("sub", "nested.md"))
}

func TestScan_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	docs, err := Scan([]config.Directory{{Path: dir}})
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestScan_MultipleDirectories(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	writeFile(t, dir1, "a.md", "# A")
	writeFile(t, dir2, "b.md", "# B")

	docs, err := Scan([]config.Directory{{Path: dir1}, {Path: dir2}})
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestScan_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plain.md", "Just some text without frontmatter.")

	docs, err := Scan([]config.Directory{{Path: dir}})
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "plain", docs[0].Title)
	assert.Contains(t, docs[0].Contents, "Just some text")
}

func writeFile(t *testing.T, base, rel, content string) {
	t.Helper()
	path := filepath.Join(base, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
