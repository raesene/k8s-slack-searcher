package indexer

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/raesene/k8s-slack-searcher/pkg/database"
	"github.com/raesene/k8s-slack-searcher/pkg/models"
)

type Indexer struct {
	db           *database.DB
	sourceDir    string
	channelName  string
	totalFiles   int
	processedFiles int
}

// NewIndexer creates a new indexer for a given channel directory
func NewIndexer(sourceDir, channelName string) (*Indexer, error) {
	db, err := database.NewDB(channelName)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return &Indexer{
		db:          db,
		sourceDir:   sourceDir,
		channelName: channelName,
	}, nil
}

// Close closes the indexer and database connection
func (idx *Indexer) Close() error {
	return idx.db.Close()
}

// IndexChannel indexes all data for a specific channel
func (idx *Indexer) IndexChannel() error {
	fmt.Printf("Indexing channel: %s\n", idx.channelName)

	// First, load users and channels data
	if err := idx.loadUsers(); err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	if err := idx.loadChannels(); err != nil {
		return fmt.Errorf("failed to load channels: %w", err)
	}

	// Then process message files in the channel directory
	channelDir := filepath.Join(idx.sourceDir, idx.channelName)
	if err := idx.processMessageFiles(channelDir); err != nil {
		return fmt.Errorf("failed to process message files: %w", err)
	}

	// Print completion statistics
	stats, err := idx.db.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("Indexing complete!\n")
	fmt.Printf("- Users: %d\n", stats["users"])
	fmt.Printf("- Channels: %d\n", stats["channels"])
	fmt.Printf("- Messages: %d\n", stats["messages"])
	fmt.Printf("- Files processed: %d\n", idx.processedFiles)

	return nil
}

// loadUsers loads users from users.json
func (idx *Indexer) loadUsers() error {
	usersFile := filepath.Join(idx.sourceDir, "users.json")
	
	data, err := os.ReadFile(usersFile)
	if err != nil {
		return fmt.Errorf("failed to read users.json: %w", err)
	}

	var usersJSON []models.UserJSON
	if err := json.Unmarshal(data, &usersJSON); err != nil {
		return fmt.Errorf("failed to parse users.json: %w", err)
	}

	fmt.Printf("Loading %d users...\n", len(usersJSON))

	for _, userJSON := range usersJSON {
		user := &models.User{
			ID:          userJSON.ID,
			Name:        userJSON.Name,
			RealName:    userJSON.Profile.RealName,
			DisplayName: userJSON.Profile.DisplayName,
			IsBot:       userJSON.IsBot,
			Deleted:     userJSON.Deleted,
		}

		if err := idx.db.InsertUser(user); err != nil {
			return fmt.Errorf("failed to insert user %s: %w", user.ID, err)
		}
	}

	return nil
}

// loadChannels loads channels from channels.json
func (idx *Indexer) loadChannels() error {
	channelsFile := filepath.Join(idx.sourceDir, "channels.json")
	
	data, err := os.ReadFile(channelsFile)
	if err != nil {
		return fmt.Errorf("failed to read channels.json: %w", err)
	}

	var channels []models.Channel
	if err := json.Unmarshal(data, &channels); err != nil {
		return fmt.Errorf("failed to parse channels.json: %w", err)
	}

	fmt.Printf("Loading %d channels...\n", len(channels))

	for _, channel := range channels {
		if err := idx.db.InsertChannel(&channel); err != nil {
			return fmt.Errorf("failed to insert channel %s: %w", channel.ID, err)
		}
	}

	return nil
}

// processMessageFiles processes all JSON message files in the channel directory
func (idx *Indexer) processMessageFiles(channelDir string) error {
	// Count total files first
	err := filepath.WalkDir(channelDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".json") {
			idx.totalFiles++
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to count files: %w", err)
	}

	fmt.Printf("Processing %d message files...\n", idx.totalFiles)

	// Process each JSON file
	err = filepath.WalkDir(channelDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		filename := filepath.Base(path)
		if err := idx.processMessageFile(path, filename); err != nil {
			fmt.Printf("Warning: failed to process %s: %v\n", filename, err)
		} else {
			idx.processedFiles++
			if idx.processedFiles%50 == 0 {
				fmt.Printf("Processed %d/%d files...\n", idx.processedFiles, idx.totalFiles)
			}
		}

		return nil
	})

	return err
}

// processMessageFile processes a single message file
func (idx *Indexer) processMessageFile(filepath, filename string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var messages []json.RawMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Parse date from filename (format: YYYY-MM-DD.json)
	dateStr := strings.TrimSuffix(filename, ".json")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("failed to parse date from filename %s: %w", filename, err)
	}

	for _, rawMsg := range messages {
		var msgMap map[string]interface{}
		if err := json.Unmarshal(rawMsg, &msgMap); err != nil {
			continue // Skip malformed messages
		}

		// Only process human messages (skip bot messages and system messages)
		if subtype, ok := msgMap["subtype"].(string); ok {
			if subtype == "bot_message" {
				continue
			}
		}

		// Skip messages without user ID or text
		userID, hasUser := msgMap["user"].(string)
		text, hasText := msgMap["text"].(string)
		if !hasUser || !hasText || strings.TrimSpace(text) == "" {
			continue
		}

		// Parse timestamp to get time of day
		timestamp, _ := msgMap["ts"].(string)
		msgType, _ := msgMap["type"].(string)
		subtype, _ := msgMap["subtype"].(string)

		// Create message with parsed timestamp
		msgTime := date
		if timestamp != "" {
			if ts, err := parseSlackTimestamp(timestamp); err == nil {
				msgTime = ts
			}
		}

		message := &models.Message{
			UserID:    userID,
			Text:      text,
			Type:      msgType,
			Subtype:   subtype,
			Timestamp: timestamp,
			Date:      msgTime,
			Filename:  filename,
		}

		if err := idx.db.InsertMessage(message); err != nil {
			return fmt.Errorf("failed to insert message: %w", err)
		}
	}

	return nil
}

// parseSlackTimestamp converts Slack timestamp to time.Time
func parseSlackTimestamp(ts string) (time.Time, error) {
	// Slack timestamps are Unix timestamps with microseconds
	// Format: "1565852586.087600"
	parts := strings.Split(ts, ".")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid timestamp format")
	}

	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(seconds, 0), nil
}