// Package main is the entry point for the qq CLI.
package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "qq",
	Short: "Local markdown search engine",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/qq/config.yaml)")
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// fatal logs a runtime error and exits. Use for errors that are not caused by
// invalid flags or arguments (those should be returned from PreRunE instead).
func fatal(err error) {
	slog.Error("command failed", "err", err)
	os.Exit(1)
}
