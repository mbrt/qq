package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mbrt/qq/internal/app"
	"github.com/mbrt/qq/internal/serve"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Run:   runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(_ *cobra.Command, _ []string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	idx, cfg, err := app.LoadAndIndex(ctx, cfgFile)
	cobra.CheckErr(err)
	defer idx.Close()

	wd, _ := os.Getwd()
	uiPath := filepath.Join(wd, "internal", "serve", "ui")
	s, err := serve.NewServer(idx, uiPath, cfg.Directories)
	cobra.CheckErr(err)
	slog.Info(fmt.Sprintf("Starting server on http://localhost:%d", servePort))
	if err := s.Serve(ctx, servePort); err != nil {
		fatal(err)
	}
}
