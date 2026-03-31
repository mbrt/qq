package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/mbrt/qq/internal/app"
	"github.com/mbrt/qq/internal/serve"
)

var (
	servePort      int
	serveNoBrowser bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Run:   runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "port to listen on")
	serveCmd.Flags().BoolVar(&serveNoBrowser, "no-browser", false, "do not open the browser automatically")
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

	onReady := func(addr string) {
		url := "http://localhost" + addr
		slog.Info(fmt.Sprintf("Starting server on %s", url))
		if !serveNoBrowser {
			if err := openBrowser(url); err != nil {
				slog.Warn("Failed to open browser", "err", err)
			}
		}
	}
	if err := s.Serve(ctx, servePort, onReady); err != nil {
		fatal(err)
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
}
