package main

import (
	"log"
	"os"

	"k8s-slack-searcher/cmd"

	"github.com/spf13/cobra"
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

func init() {
	// Add commands
	rootCmd.AddCommand(cmd.IngestCmd)
	rootCmd.AddCommand(cmd.SearchCmd)
	rootCmd.AddCommand(cmd.ListCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}