package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"k8s-slack-searcher/pkg/models"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn     *sql.DB
	filename string
}

// NewDB creates a new database connection
func NewDB(channelName string) (*DB, error) {
	// Sanitize channel name for filename
	filename := sanitizeFilename(channelName) + ".db"
	
	// Ensure databases directory exists
	dbPath := filepath.Join("databases", filename)
	
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{
		conn:     conn,
		filename: filename,
	}

	if err := db.createTables(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// sanitizeFilename removes problematic characters from channel names
func sanitizeFilename(name string) string {
	// Replace problematic characters with underscores
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

// createTables creates the necessary tables and FTS index
func (db *DB) createTables() error {
	queries := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			real_name TEXT,
			display_name TEXT,
			is_bot BOOLEAN DEFAULT FALSE,
			deleted BOOLEAN DEFAULT FALSE
		)`,
		
		// Channels table
		`CREATE TABLE IF NOT EXISTS channels (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created INTEGER,
			creator TEXT,
			is_archived BOOLEAN DEFAULT FALSE
		)`,
		
		// Messages table
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT,
			text TEXT NOT NULL,
			type TEXT,
			subtype TEXT,
			timestamp TEXT,
			date DATETIME,
			filename TEXT,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`,
		
		// FTS virtual table for full-text search
		`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts4(
			text,
			user_name,
			user_real_name,
			filename
		)`,
		
		// Trigger to keep FTS table in sync
		`CREATE TRIGGER IF NOT EXISTS messages_fts_insert AFTER INSERT ON messages BEGIN
			INSERT INTO messages_fts(rowid, text, user_name, user_real_name, filename)
			SELECT 
				new.id,
				new.text,
				COALESCE(u.name, ''),
				COALESCE(u.real_name, ''),
				new.filename
			FROM users u WHERE u.id = new.user_id;
		END`,
		
		`CREATE TRIGGER IF NOT EXISTS messages_fts_delete AFTER DELETE ON messages BEGIN
			DELETE FROM messages_fts WHERE rowid = old.id;
		END`,
		
		`CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON messages BEGIN
			DELETE FROM messages_fts WHERE rowid = old.id;
			INSERT INTO messages_fts(rowid, text, user_name, user_real_name, filename)
			SELECT 
				new.id,
				new.text,
				COALESCE(u.name, ''),
				COALESCE(u.real_name, ''),
				new.filename
			FROM users u WHERE u.id = new.user_id;
		END`,
		
		// Indexes for better performance
		`CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_date ON messages(date)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_filename ON messages(filename)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s: %w", query, err)
		}
	}

	return nil
}

// InsertUser inserts a user into the database
func (db *DB) InsertUser(user *models.User) error {
	query := `INSERT OR REPLACE INTO users (id, name, real_name, display_name, is_bot, deleted)
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	_, err := db.conn.Exec(query, user.ID, user.Name, user.RealName, user.DisplayName, user.IsBot, user.Deleted)
	return err
}

// InsertChannel inserts a channel into the database
func (db *DB) InsertChannel(channel *models.Channel) error {
	query := `INSERT OR REPLACE INTO channels (id, name, created, creator, is_archived)
			  VALUES (?, ?, ?, ?, ?)`
	
	_, err := db.conn.Exec(query, channel.ID, channel.Name, channel.Created, channel.Creator, channel.IsArchived)
	return err
}

// InsertMessage inserts a message into the database
func (db *DB) InsertMessage(message *models.Message) error {
	query := `INSERT INTO messages (user_id, text, type, subtype, timestamp, date, filename)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	
	_, err := db.conn.Exec(query, message.UserID, message.Text, message.Type, message.Subtype, 
						  message.Timestamp, message.Date, message.Filename)
	return err
}

// SearchMessages performs full-text search on messages
func (db *DB) SearchMessages(query string, limit int) ([]*models.SearchResult, error) {
	sqlQuery := `
		SELECT 
			m.id,
			m.user_id,
			m.text,
			m.type,
			m.subtype,
			m.timestamp,
			m.date,
			m.filename,
			COALESCE(u.name, '') as user_name,
			COALESCE(u.real_name, '') as user_real_name,
			0.0 as rank,
			snippet(messages_fts, '<mark>', '</mark>', '...', -1, 32) as snippet
		FROM messages_fts fts
		JOIN messages m ON m.id = fts.rowid
		LEFT JOIN users u ON u.id = m.user_id
		WHERE messages_fts MATCH ?
		LIMIT ?`

	rows, err := db.conn.Query(sqlQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []*models.SearchResult
	for rows.Next() {
		result := &models.SearchResult{}
		err := rows.Scan(
			&result.ID,
			&result.UserID,
			&result.Text,
			&result.Type,
			&result.Subtype,
			&result.Timestamp,
			&result.Date,
			&result.Filename,
			&result.UserName,
			&result.UserRealName,
			&result.Rank,
			&result.Snippet,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// GetStats returns basic statistics about the database
func (db *DB) GetStats() (map[string]int, error) {
	stats := make(map[string]int)
	
	queries := map[string]string{
		"users":    "SELECT COUNT(*) FROM users",
		"channels": "SELECT COUNT(*) FROM channels", 
		"messages": "SELECT COUNT(*) FROM messages",
	}
	
	for key, query := range queries {
		var count int
		err := db.conn.QueryRow(query).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s count: %w", key, err)
		}
		stats[key] = count
	}
	
	return stats, nil
}