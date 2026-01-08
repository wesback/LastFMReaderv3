package main

import (
	"fmt"
	"os"

	"github.com/lastfm-reader/lastfm-sync/cmd/lastfm-sync/commands"
	"github.com/lastfm-reader/lastfm-sync/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lastfm-sync",
	Short: "Last.fm scrobble sync tool with incremental updates",
	Long: `lastfm-sync is a CLI tool for exporting Last.fm scrobble history.

Supports local NDJSON output and Azure Blob Storage with incremental sync,
rate limiting, and crash-safe resumption.`,
	Version: version.String(),
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(commands.FetchCommand())
	rootCmd.AddCommand(commands.MergeCommand())
	rootCmd.AddCommand(commands.NormalizeCommand())
	// TODO: Add show-watermark, set-watermark commands
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
