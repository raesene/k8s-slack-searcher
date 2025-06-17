package main

import (
	"fmt"
	"log"
	"os"

	"github.com/raesene/k8s-slack-searcher/cmd"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "k8s-slack-searcher",
	Short: "Search through Kubernetes Slack workspace archives",
	Long: `A tool to index and search through Slack workspace archives.
	
It can ingest channel data and create searchable databases,
then provide full-text search capabilities across the indexed content.

Commands:
  ingest <channel>  Index a channel directory and create a database
  search <query>    Search messages in a channel database
  list              List available databases`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("k8s-slack-searcher %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Date: %s\n", date)
	},
}

func init() {
	// Add commands
	rootCmd.AddCommand(cmd.IngestCmd)
	rootCmd.AddCommand(cmd.SearchCmd)
	rootCmd.AddCommand(cmd.ListCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}