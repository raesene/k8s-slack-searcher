package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raesene/k8s-slack-searcher/pkg/searcher"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search messages in a channel database",
	Long: `Search for messages in a channel database using full-text search.
	
The search supports SQLite FTS4 syntax including quoted phrases, 
boolean operators (AND, OR, NOT), and prefix matching.

Examples:
  k8s-slack-searcher search "authentication" --database sig-auth
  k8s-slack-searcher search "cert* AND rotate*" --database sig-auth
  k8s-slack-searcher search "RBAC OR authentication" --database sig-auth --html search_results.html`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available databases",
	Long:  `List all available channel databases that can be searched.`,
	RunE:  runList,
}

var (
	databaseName string
	searchLimit  int
	showStats    bool
	htmlOutput   string
)

func init() {
	searchCmd.Flags().StringVarP(&databaseName, "database", "d", "", 
		"Database name (channel name) to search in (required)")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 10, 
		"Maximum number of results to return")
	searchCmd.Flags().BoolVar(&showStats, "stats", false, 
		"Show database statistics")
	searchCmd.Flags().StringVar(&htmlOutput, "html", "", 
		"Generate HTML output file with thread context (e.g., --html results.html)")
	
	searchCmd.MarkFlagRequired("database")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	
	// Validate database exists
	if !searcher.ValidateDatabaseExists(databaseName) {
		return fmt.Errorf("database not found: %s. Run 'k8s-slack-searcher list' to see available databases", databaseName)
	}
	
	// Create searcher
	search, err := searcher.NewSearcher(databaseName)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer search.Close()
	
	// Show stats if requested
	if showStats {
		stats, err := search.GetStats()
		if err != nil {
			return fmt.Errorf("failed to get stats: %w", err)
		}
		
		fmt.Printf("Database: %s\n", databaseName)
		fmt.Printf("- Users: %d\n", stats["users"])
		fmt.Printf("- Channels: %d\n", stats["channels"])
		fmt.Printf("- Messages: %d\n\n", stats["messages"])
	}
	
	// Perform search
	fmt.Printf("Searching for: %s\n", query)
	fmt.Printf("Database: %s\n", databaseName)
	fmt.Printf("Limit: %d\n", searchLimit)
	
	if htmlOutput != "" {
		fmt.Printf("HTML output: %s\n", htmlOutput)
	}
	fmt.Print("\n")
	
	results, err := search.Search(query, searchLimit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	
	// Generate HTML output if requested
	if htmlOutput != "" {
		// Create output directory if needed
		if dir := filepath.Dir(htmlOutput); dir != "." {
			if err := ensureDir(dir); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}
		
		// Generate HTML file
		err := search.GenerateHTMLOutput(results, query, databaseName, htmlOutput)
		if err != nil {
			return fmt.Errorf("failed to generate HTML output: %w", err)
		}
		
		fmt.Printf("HTML output generated: %s\n", htmlOutput)
		
		// Also show text output for immediate feedback
		if len(results) > 0 {
			fmt.Printf("Found %d result(s) with thread context.\n", len(results))
		} else {
			fmt.Println("No results found.")
		}
	} else {
		// Format and display results in text format
		output := searcher.FormatResults(results)
		fmt.Print(output)
	}
	
	return nil
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(dir string) error {
	if strings.TrimSpace(dir) == "" || dir == "." {
		return nil
	}
	
	return os.MkdirAll(dir, 0755)
}

func runList(cmd *cobra.Command, args []string) error {
	databases, err := searcher.ListDatabases()
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}
	
	if len(databases) == 0 {
		fmt.Println("No databases found. Use 'ingest' command to create a database first.")
		return nil
	}
	
	fmt.Printf("Available databases (%d):\n\n", len(databases))
	for _, db := range databases {
		fmt.Printf("  %s\n", db)
	}
	
	fmt.Printf("\nUse 'k8s-slack-searcher search <query> --database <name>' to search.\n")
	
	return nil
}