package searcher

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s-slack-searcher/pkg/database"
	"k8s-slack-searcher/pkg/models"
)

type Searcher struct {
	db *database.DB
}

// NewSearcher creates a new searcher for a specific database
func NewSearcher(channelName string) (*Searcher, error) {
	db, err := database.NewDB(channelName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Searcher{db: db}, nil
}

// Close closes the searcher and database connection
func (s *Searcher) Close() error {
	return s.db.Close()
}

// Search performs a full-text search and returns formatted results
func (s *Searcher) Search(query string, limit int) ([]*models.SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	return s.db.SearchMessages(query, limit)
}

// GetStats returns database statistics
func (s *Searcher) GetStats() (map[string]int, error) {
	return s.db.GetStats()
}

// FormatResults formats search results for display
func FormatResults(results []*models.SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}

	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("Found %d result(s):\n\n", len(results)))
	
	for i, result := range results {
		// Parse date for display
		date := result.Date.Format("2006-01-02 15:04:05")
		
		// Determine user display name
		userName := result.UserName
		if result.UserRealName != "" {
			userName = fmt.Sprintf("%s (%s)", result.UserRealName, result.UserName)
		}
		if userName == "" {
			userName = result.UserID
		}
		
		// Format message
		output.WriteString(fmt.Sprintf("--- Result %d ---\n", i+1))
		output.WriteString(fmt.Sprintf("User: %s\n", userName))
		output.WriteString(fmt.Sprintf("Date: %s\n", date))
		output.WriteString(fmt.Sprintf("File: %s\n", result.Filename))
		
		// Show snippet if available, otherwise show full text
		messageText := result.Text
		if result.Snippet != "" {
			messageText = result.Snippet
		}
		
		// Clean up the message text
		messageText = strings.ReplaceAll(messageText, "\n", " ")
		if len(messageText) > 500 {
			messageText = messageText[:497] + "..."
		}
		
		output.WriteString(fmt.Sprintf("Message: %s\n\n", messageText))
	}
	
	return output.String()
}

// ValidateDatabaseExists checks if a database file exists for the given channel
func ValidateDatabaseExists(channelName string) bool {
	// Sanitize filename same way as database package
	filename := sanitizeFilename(channelName) + ".db"
	dbPath := filepath.Join("databases", filename)
	
	return fileExists(dbPath)
}

// ListDatabases lists all available database files
func ListDatabases() ([]string, error) {
	pattern := filepath.Join("databases", "*.db")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	
	var databases []string
	for _, match := range matches {
		// Extract just the filename without extension
		base := filepath.Base(match)
		name := strings.TrimSuffix(base, ".db")
		databases = append(databases, name)
	}
	
	return databases, nil
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := filepath.Abs(filename)
	return err == nil
}

// sanitizeFilename removes problematic characters from channel names
// This should match the implementation in database package
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		":", "_",
		"/", "_",
		"\\", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}