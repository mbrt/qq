package serve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mbrt/qq/internal/config"
	"github.com/mbrt/qq/internal/document"
	"github.com/mbrt/qq/internal/index"
)

func TestHandleFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "photo.png"), []byte("fake-png"), 0o644))

	srv := newTestServer(t, []string{dir}, nil)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "existing file",
			path:       "/files/0/sub/photo.png",
			wantStatus: http.StatusOK,
			wantBody:   "fake-png",
		},
		{
			name:       "missing file",
			path:       "/files/0/nonexistent.png",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "directory listing is empty",
			path:       "/files/0/sub/",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid dir index",
			path:       "/files/99/sub/photo.png",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			srv.mux.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestHandleFile_MultipleDirs(t *testing.T) {
	dir0 := t.TempDir()
	dir1 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir0, "a.png"), []byte("from-dir0"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "b.png"), []byte("from-dir1"), 0o644))

	srv := newTestServer(t, []string{dir0, dir1}, nil)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "file from dir 0",
			path:       "/files/0/a.png",
			wantStatus: http.StatusOK,
			wantBody:   "from-dir0",
		},
		{
			name:       "file from dir 1",
			path:       "/files/1/b.png",
			wantStatus: http.StatusOK,
			wantBody:   "from-dir1",
		},
		{
			name:       "dir 0 does not serve dir 1 files",
			path:       "/files/0/b.png",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "dir 1 does not serve dir 0 files",
			path:       "/files/1/a.png",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			srv.mux.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestHandleRead_NotFound(t *testing.T) {
	srv := newTestServer(t, []string{t.TempDir()}, nil)

	req := httptest.NewRequest("GET", "/read/nonexistent.md", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleRead_WithLocalImage(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "2026", "article.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(mdPath), 0o755))
	require.NoError(t, os.WriteFile(mdPath, []byte("---\ntitle: Test\n---\n\n![photo](photo.png)\n"), 0o644))

	imgPath := filepath.Join(dir, "2026", "photo.png")
	require.NoError(t, os.WriteFile(imgPath, []byte("fake-png"), 0o644))

	docs := []document.Document{
		{
			ID:       "2026/article.md",
			Title:    "Test",
			Contents: "![photo](photo.png)\n",
			Updated:  time.Now(),
		},
	}
	srv := newTestServer(t, []string{dir}, docs)

	req := httptest.NewRequest("GET", "/read/2026/article.md", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `src="/files/0/2026/photo.png"`)

	req = httptest.NewRequest("GET", "/files/0/2026/photo.png", nil)
	w = httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "fake-png", w.Body.String())
}

func newTestServer(t *testing.T, docDirs []string, docs []document.Document) *Server {
	t.Helper()
	idxDir := filepath.Join(t.TempDir(), "idx")
	idx, err := index.Open(idxDir)
	require.NoError(t, err)
	t.Cleanup(func() { idx.Close() })

	if docs != nil {
		_, err = idx.Reconcile(context.Background(), docs)
		require.NoError(t, err)
	}

	var dirs []config.Directory
	for _, d := range docDirs {
		dirs = append(dirs, config.Directory{Path: d})
	}
	srv, err := NewServer(idx, dirs)
	require.NoError(t, err)
	return srv
}
