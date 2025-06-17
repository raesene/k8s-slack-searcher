package searcher

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/raesene/k8s-slack-searcher/pkg/database"
	"github.com/raesene/k8s-slack-searcher/pkg/models"
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

// GetThreadMessages retrieves all messages in a thread
func (s *Searcher) GetThreadMessages(threadTS string) ([]*models.Message, error) {
	return s.db.GetThreadMessages(threadTS)
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

// ThreadedSearchResult represents a search result with its thread context
type ThreadedSearchResult struct {
	OriginalResult *models.SearchResult
	ThreadMessages []*models.Message
	IsThreaded     bool
}

// GenerateHTMLOutput creates an HTML file with search results and thread context
func (s *Searcher) GenerateHTMLOutput(results []*models.SearchResult, query, channelName, outputPath string) error {
	// Collect thread context for each result
	var threadedResults []*ThreadedSearchResult
	
	for _, result := range results {
		threadedResult := &ThreadedSearchResult{
			OriginalResult: result,
			IsThreaded:     false,
		}
		
		// Check if this message is part of a thread
		var threadTS string
		if result.ThreadTS != "" {
			threadTS = result.ThreadTS
		} else if result.ReplyCount > 0 {
			// This is a parent message with replies
			threadTS = result.Timestamp
		}
		
		if threadTS != "" {
			threadMessages, err := s.GetThreadMessages(threadTS)
			if err == nil && len(threadMessages) > 1 {
				threadedResult.ThreadMessages = threadMessages
				threadedResult.IsThreaded = true
			}
		}
		
		threadedResults = append(threadedResults, threadedResult)
	}
	
	// Create HTML content
	htmlContent, err := generateHTML(threadedResults, query, channelName)
	if err != nil {
		return fmt.Errorf("failed to generate HTML: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(outputPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}
	
	return nil
}

// generateHTML creates the HTML content using templates
func generateHTML(results []*ThreadedSearchResult, query, channelName string) (string, error) {
	tmpl := template.Must(template.New("search_results").Funcs(template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("January 2, 2006 at 3:04 PM")
		},
		"formatUser": func(realName, userName, userID string) string {
			if realName != "" && userName != "" {
				return fmt.Sprintf("%s (%s)", realName, userName)
			}
			if userName != "" {
				return userName
			}
			return userID
		},
		"safeHTML": func(text string) template.HTML {
			// Convert newlines to <br> and preserve HTML marks from search snippets
			text = strings.ReplaceAll(text, "\n", "<br>")
			return template.HTML(text)
		},
		"truncate": func(text string, length int) string {
			if len(text) <= length {
				return text
			}
			return text[:length] + "..."
		},
	}).Parse(htmlTemplate))
	
	data := struct {
		Query       string
		ChannelName string
		Results     []*ThreadedSearchResult
		ResultCount int
		Timestamp   string
	}{
		Query:       query,
		ChannelName: channelName,
		Results:     results,
		ResultCount: len(results),
		Timestamp:   time.Now().Format("January 2, 2006 at 3:04 PM"),
	}
	
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Search Results: {{.Query}} - {{.ChannelName}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f8f9fa;
            color: #333;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            padding: 30px;
        }
        .header {
            border-bottom: 2px solid #e9ecef;
            padding-bottom: 20px;
            margin-bottom: 30px;
        }
        .header h1 {
            margin: 0 0 10px 0;
            color: #2c3e50;
        }
        .search-info {
            color: #6c757d;
            font-size: 0.9em;
        }
        .result {
            border: 1px solid #e9ecef;
            border-radius: 8px;
            margin-bottom: 30px;
            background: #fff;
        }
        .result-header {
            background: #f8f9fa;
            padding: 15px 20px;
            border-bottom: 1px solid #e9ecef;
            border-radius: 8px 8px 0 0;
        }
        .result-meta {
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
        }
        .user-info {
            font-weight: 600;
            color: #495057;
        }
        .date-info {
            color: #6c757d;
            font-size: 0.9em;
        }
        .message-content {
            padding: 20px;
        }
        .original-message {
            background: #e3f2fd;
            border-left: 4px solid #2196f3;
            padding: 15px;
            margin-bottom: 15px;
            border-radius: 4px;
        }
        .thread-section {
            margin-top: 20px;
        }
        .thread-header {
            font-weight: 600;
            color: #495057;
            margin-bottom: 15px;
            padding-bottom: 5px;
            border-bottom: 1px solid #e9ecef;
        }
        .thread-message {
            background: #f8f9fa;
            border-left: 3px solid #6c757d;
            padding: 12px 15px;
            margin-bottom: 10px;
            border-radius: 4px;
        }
        .thread-message.parent {
            background: #fff3cd;
            border-left-color: #ffc107;
        }
        .thread-user {
            font-weight: 600;
            color: #495057;
            margin-bottom: 5px;
        }
        .thread-date {
            font-size: 0.8em;
            color: #6c757d;
            margin-bottom: 8px;
        }
        .thread-text {
            line-height: 1.5;
        }
        mark {
            background-color: #ffeb3b;
            padding: 2px 4px;
            border-radius: 3px;
        }
        .no-results {
            text-align: center;
            color: #6c757d;
            font-style: italic;
            padding: 40px;
        }
        .file-info {
            font-size: 0.8em;
            color: #6c757d;
            margin-top: 5px;
        }
        @media (max-width: 768px) {
            body {
                padding: 10px;
            }
            .container {
                padding: 15px;
            }
            .result-meta {
                flex-direction: column;
                align-items: flex-start;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Search Results</h1>
            <div class="search-info">
                <strong>Query:</strong> "{{.Query}}" in <strong>#{{.ChannelName}}</strong><br>
                <strong>Results:</strong> {{.ResultCount}} found<br>
                <strong>Generated:</strong> {{.Timestamp}}
            </div>
        </div>

        {{if .Results}}
            {{range $i, $result := .Results}}
            <div class="result">
                <div class="result-header">
                    <div class="result-meta">
                        <div class="user-info">
                            {{formatUser $result.OriginalResult.UserRealName $result.OriginalResult.UserName $result.OriginalResult.UserID}}
                        </div>
                        <div class="date-info">
                            {{formatDate $result.OriginalResult.Date}}
                            <div class="file-info">{{$result.OriginalResult.Filename}}</div>
                        </div>
                    </div>
                </div>
                
                <div class="message-content">
                    <div class="original-message">
                        <strong>Original Message:</strong><br>
                        {{if $result.OriginalResult.Snippet}}
                            {{safeHTML $result.OriginalResult.Snippet}}
                        {{else}}
                            {{safeHTML (truncate $result.OriginalResult.Text 500)}}
                        {{end}}
                    </div>

                    {{if $result.IsThreaded}}
                    <div class="thread-section">
                        <div class="thread-header">
                            🧵 Thread Context ({{len $result.ThreadMessages}} messages)
                        </div>
                        {{range $msg := $result.ThreadMessages}}
                        <div class="thread-message {{if eq $msg.Timestamp $msg.ThreadTS}}parent{{end}}">
                            <div class="thread-user">
                                {{formatUser $msg.UserRealName $msg.UserName $msg.UserID}}
                                {{if eq $msg.Timestamp $msg.ThreadTS}}(thread starter){{end}}
                            </div>
                            <div class="thread-date">{{formatDate $msg.Date}}</div>
                            <div class="thread-text">{{safeHTML (truncate $msg.Text 300)}}</div>
                        </div>
                        {{end}}
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        {{else}}
            <div class="no-results">
                No results found for your search query.
            </div>
        {{end}}
    </div>
</body>
</html>`

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