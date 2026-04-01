package serve

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/mbrt/qq/internal/index"
	tmplpkg "github.com/mbrt/qq/internal/serve/template"
	"github.com/mbrt/qq/internal/timeutil"
)

var tmplFuncs = template.FuncMap{
	"toTimeAgo": func(v any) string {
		if t, ok := v.(time.Time); ok && !t.IsZero() {
			return timeutil.TimeAgo(time.Now(), t)
		}
		return ""
	},
	"displayDuration": func(v any) string {
		if d, ok := v.(time.Duration); ok {
			return d.Round(time.Millisecond).String()
		}
		return fmt.Sprintf("%v", v)
	},
	"join": func(sep string, items []string) string {
		return strings.Join(items, sep)
	},
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"oneWeekAgo": func() string {
		return time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	},
}

func newDynamicHandler(api *apiHandler, tfs fs.FS) (dynamicHandler, error) {
	r, err := tmplpkg.New(tfs, tmplFuncs)
	if err != nil {
		return dynamicHandler{}, fmt.Errorf("creating renderer: %w", err)
	}
	return dynamicHandler{api: api, renderer: r}, nil
}

type dynamicHandler struct {
	api      *apiHandler
	renderer renderer
}

func (d dynamicHandler) Home(_ context.Context, w io.Writer) error {
	return d.renderer.Render(w, "home.html", nil)
}

func (d dynamicHandler) Search(ctx context.Context, w io.Writer, query string) error {
	res, err := d.api.Search(ctx, query)
	if err != nil {
		return err
	}
	data := searchViewModel{
		SearchResult: res,
		Query:        query,
	}
	return d.renderer.Render(w, "search.html", data)
}

func (d dynamicHandler) Read(ctx context.Context, w io.Writer, id string) error {
	res, err := d.api.Read(ctx, id)
	if err != nil {
		return err
	}
	data := readViewModel{
		ReadResult: res,
	}
	return d.renderer.Render(w, "read.html", data)
}

type renderer interface {
	Render(w io.Writer, name string, data any) error
}

type readViewModel struct {
	index.ReadResult
	Query string
}

type searchViewModel struct {
	index.SearchResult
	Query string
}
