// Package template provides HTML template rendering from an fs.FS root.
package template

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
)

// New creates a Templater by parsing all templates in the root filesystem.
//
// Expected structure:
//
//	root/
//	  home.html
//	  search.html
//	  read.html
//	  partials/
//	    head.html
//	    header.html
//	    footer.html
func New(root fs.FS, fmap template.FuncMap) (*Templater, error) {
	partials := template.New("").Funcs(fmap)

	partPaths, err := fs.Glob(root, "partials/*")
	if err != nil {
		return nil, fmt.Errorf("globbing partials: %w", err)
	}
	for _, p := range partPaths {
		t, err := addTemplate(partials, root, p)
		if err != nil {
			return nil, fmt.Errorf("parsing partial %q: %w", p, err)
		}
		partials = t
	}

	rootPaths, err := fs.Glob(root, "*.html")
	if err != nil {
		return nil, fmt.Errorf("globbing root templates: %w", err)
	}
	tmpls := make(map[string]*template.Template)
	for _, rt := range rootPaths {
		newRoot, err := partials.Clone()
		if err != nil {
			return nil, fmt.Errorf("cloning partials: %w", err)
		}
		newRoot, err = addTemplate(newRoot, root, rt)
		if err != nil {
			return nil, fmt.Errorf("parsing root template %q: %w", rt, err)
		}
		tmpls[rt] = newRoot
	}

	return &Templater{ts: tmpls}, nil
}

func addTemplate(tmpl *template.Template, root fs.FS, tpath string) (*template.Template, error) {
	f, err := root.Open(tpath)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", tpath, err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", tpath, err)
	}
	name := path.Base(tpath)
	text := fmt.Sprintf("{{define %q}}%s{{end}}", name, string(b))
	return tmpl.Parse(text)
}

// Templater holds parsed templates and renders them by name.
type Templater struct {
	ts map[string]*template.Template
}

// Render executes the named template with the given data, writing to w.
func (t *Templater) Render(w io.Writer, name string, data any) error {
	tmpl, ok := t.ts[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	return tmpl.ExecuteTemplate(w, name, data)
}
