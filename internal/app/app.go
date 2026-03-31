package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mbrt/qq/internal/config"
	"github.com/mbrt/qq/internal/index"
	"github.com/mbrt/qq/internal/scanner"
)

// LoadAndIndex loads the config, scans directories, and reconciles the index.
// The caller is responsible for closing the returned index.
func LoadAndIndex(ctx context.Context, cfgFile string) (*index.Index, config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, config.Config{}, fmt.Errorf("loading config: %w", err)
	}
	docs, err := scanner.Scan(cfg.Directories)
	if err != nil {
		return nil, cfg, fmt.Errorf("scanning directories: %w", err)
	}
	slog.Info("Scanned files", "documents", len(docs))

	idx, err := index.Open(cfg.IndexPath)
	if err != nil {
		return nil, cfg, fmt.Errorf("opening index: %w", err)
	}
	stats, err := idx.Reconcile(ctx, docs)
	if err != nil {
		idx.Close()
		return nil, cfg, fmt.Errorf("reconciling index: %w", err)
	}
	slog.Info("Index reconciled",
		"added", stats.Added,
		"updated", stats.Updated,
		"removed", stats.Removed,
		"unchanged", stats.Unchanged)

	return idx, cfg, nil
}
