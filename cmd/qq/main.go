// Package main is the entry point for the qq CLI.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/mbrt/qq/internal/app"
	"github.com/mbrt/qq/internal/index"
	"github.com/mbrt/qq/internal/serve"
	"github.com/mbrt/qq/internal/timeutil"
)

var (
	cfgFile      string
	servePort    int
	searchFormat string

	rootCmd = &cobra.Command{
		Use:   "qq",
		Short: "Local markdown search engine",
	}
	searchCmd = &cobra.Command{
		Use:   "search [query]",
		Short: "Search markdown files",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runSearch,
	}
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the web server",
		RunE:  runServe,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/qq/config.yaml)")
	searchCmd.Flags().StringVarP(&searchFormat, "format", "f", "pretty", "output format: pretty, path, json")
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "port to listen on")
	rootCmd.AddCommand(searchCmd, serveCmd)
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSearch(_ *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	idx, _, err := app.LoadAndIndex(ctx, cfgFile)
	if err != nil {
		return err
	}
	defer idx.Close()

	query := strings.Join(args, " ")
	results, err := idx.Search(query)
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	switch searchFormat {
	case "pretty":
		printSearchPretty(os.Stdout, results)
	case "path":
		printSearchPaths(os.Stdout, results)
	case "json":
		return printSearchJSON(os.Stdout, results)
	default:
		return fmt.Errorf("unknown format %q (valid: pretty, path, json)", searchFormat)
	}
	return nil
}

func runServe(_ *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	idx, cfg, err := app.LoadAndIndex(ctx, cfgFile)
	if err != nil {
		return err
	}
	defer idx.Close()

	wd, _ := os.Getwd()
	uiPath := filepath.Join(wd, "internal", "serve", "ui")
	s, err := serve.NewServer(idx, uiPath, cfg.Directories)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}
	slog.Info(fmt.Sprintf("Starting server on http://localhost:%d", servePort))
	return s.Serve(ctx, servePort)
}

func printSearchPretty(w io.Writer, results index.SearchResult) {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	faint := color.New(color.Faint)

	fmt.Fprintf(w, "%d results (%s)\n", results.Total, results.Took)
	for _, hit := range results.Hits {
		fmt.Fprintln(w)
		bold.Fprintf(w, "- %s\n", hit.Title)

		if hit.Path != "" {
			cyan.Fprintf(w, "  %s\n", formatPath(hit.Path))
		}

		var parts []string
		if hit.Source != "" {
			parts = append(parts, hit.Source)
		}
		for _, t := range hit.Tags {
			parts = append(parts, t)
		}
		if !hit.Updated.IsZero() {
			parts = append(parts, timeutil.TimeAgo(time.Now(), hit.Updated))
		}
		if len(parts) > 0 {
			faint.Fprintf(w, "  %s\n", strings.Join(parts, " | "))
		}

		if len(hit.Fragments) > 0 {
			snippet := index.StripHTMLTags(hit.Fragments[0])
			for line := range strings.SplitSeq(snippet, "\n") {
				fmt.Fprintf(w, "      %s\n", line)
			}
		}
	}
}

func printSearchPaths(w io.Writer, results index.SearchResult) {
	for _, hit := range results.Hits {
		if hit.Path != "" {
			fmt.Fprintln(w, hit.Path)
		}
	}
}

type jsonOutput struct {
	Total int             `json:"total"`
	Took  string          `json:"took"`
	Hits  []jsonOutputHit `json:"hits"`
}

type jsonOutputHit struct {
	ID      string   `json:"id"`
	Score   float64  `json:"score"`
	Title   string   `json:"title"`
	Path    string   `json:"path,omitempty"`
	Source  string   `json:"source,omitempty"`
	Author  string   `json:"author,omitempty"`
	URL     string   `json:"url,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Updated string   `json:"updated,omitempty"`
	Excerpt string   `json:"excerpt,omitempty"`
}

func printSearchJSON(w io.Writer, results index.SearchResult) error {
	out := jsonOutput{
		Total: results.Total,
		Took:  results.Took.String(),
		Hits:  make([]jsonOutputHit, len(results.Hits)),
	}
	for i, hit := range results.Hits {
		var updated string
		if !hit.Updated.IsZero() {
			updated = hit.Updated.Format(time.RFC3339)
		}
		out.Hits[i] = jsonOutputHit{
			ID:      hit.ID,
			Score:   hit.Score,
			Title:   hit.Title,
			Path:    hit.Path,
			Source:  hit.Source,
			Author:  hit.Author,
			URL:     hit.URL,
			Tags:    hit.Tags,
			Updated: updated,
			Excerpt: hit.Excerpt,
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func formatPath(path string) string {
	short := shortenHome(path)
	if strings.Contains(short, " ") {
		return "'" + short + "'"
	}
	return short
}

func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if rel, ok := strings.CutPrefix(path, home); ok {
		return "~" + rel
	}
	return path
}
