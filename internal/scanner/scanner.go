package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mbrt/qq/internal/config"
	"github.com/mbrt/qq/internal/document"
)

// Scan walks all configured directories and returns parsed documents.
func Scan(dirs []config.Directory) ([]document.Document, error) {
	var docs []document.Document
	for _, dir := range dirs {
		d, err := scanDir(dir.Path)
		if err != nil {
			return nil, fmt.Errorf("scanning %q: %w", dir.Path, err)
		}
		docs = append(docs, d...)
	}
	return docs, nil
}

func scanDir(root string) ([]document.Document, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var docs []document.Document
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %q: %w", path, err)
		}
		filename := strings.TrimSuffix(d.Name(), ".md")
		doc := document.Parse(rel, filename, data, info.ModTime())
		doc.Path = path
		docs = append(docs, doc)
		return nil
	})
	return docs, err
}
