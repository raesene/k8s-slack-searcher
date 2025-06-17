package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/raesene/k8s-slack-searcher/pkg/indexer"

	"github.com/spf13/cobra"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest <channel-directory>",
	Short: "Index a Slack channel directory",
	Long: `Index a Slack channel directory and create a searchable database.
	
The channel directory should be a subdirectory within the source-data directory
containing daily JSON message files (e.g., 2019-01-15.json).

Example:
  k8s-slack-searcher ingest sig-auth`,
	Args: cobra.ExactArgs(1),
	RunE: runIngest,
}

var (
	sourceDataDir string
)

func init() {
	ingestCmd.Flags().StringVarP(&sourceDataDir, "source", "s", "source-data", 
		"Source data directory containing users.json, channels.json, and channel subdirectories")
}

func runIngest(cmd *cobra.Command, args []string) error {
	channelName := args[0]
	
	// Validate source directory exists
	if _, err := os.Stat(sourceDataDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", sourceDataDir)
	}
	
	// Validate channel directory exists
	channelDir := filepath.Join(sourceDataDir, channelName)
	if _, err := os.Stat(channelDir); os.IsNotExist(err) {
		return fmt.Errorf("channel directory does not exist: %s", channelDir)
	}
	
	// Check for required files
	usersFile := filepath.Join(sourceDataDir, "users.json")
	channelsFile := filepath.Join(sourceDataDir, "channels.json")
	
	if _, err := os.Stat(usersFile); os.IsNotExist(err) {
		return fmt.Errorf("users.json not found in source directory: %s", usersFile)
	}
	
	if _, err := os.Stat(channelsFile); os.IsNotExist(err) {
		return fmt.Errorf("channels.json not found in source directory: %s", channelsFile)
	}
	
	// Ensure databases directory exists
	if err := os.MkdirAll("databases", 0755); err != nil {
		return fmt.Errorf("failed to create databases directory: %w", err)
	}
	
	// Create and run indexer
	fmt.Printf("Creating database for channel: %s\n", channelName)
	
	idx, err := indexer.NewIndexer(sourceDataDir, channelName)
	if err != nil {
		return fmt.Errorf("failed to create indexer: %w", err)
	}
	defer idx.Close()
	
	if err := idx.IndexChannel(); err != nil {
		return fmt.Errorf("failed to index channel: %w", err)
	}
	
	fmt.Printf("\nDatabase created successfully: databases/%s.db\n", channelName)
	
	return nil
}