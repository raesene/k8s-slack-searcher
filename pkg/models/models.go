package models

import "time"

// User represents a Slack user from users.json
type User struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	RealName    string `json:"real_name" db:"real_name"`
	DisplayName string `json:"display_name" db:"display_name"`
	IsBot       bool   `json:"is_bot" db:"is_bot"`
	Deleted     bool   `json:"deleted" db:"deleted"`
}

// Profile represents the nested profile object in User
type Profile struct {
	RealName    string `json:"real_name"`
	DisplayName string `json:"display_name"`
}

// UserJSON represents the full user structure from the JSON file
type UserJSON struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Profile Profile `json:"profile"`
	IsBot   bool    `json:"is_bot"`
	Deleted bool    `json:"deleted"`
}

// Channel represents a Slack channel from channels.json
type Channel struct {
	ID         string `json:"id" db:"id"`
	Name       string `json:"name" db:"name"`
	Created    int64  `json:"created" db:"created"`
	Creator    string `json:"creator" db:"creator"`
	IsArchived bool   `json:"is_archived" db:"is_archived"`
}

// Message represents a Slack message from daily JSON files
type Message struct {
	ID        int       `db:"id"`
	UserID    string    `json:"user" db:"user_id"`
	Text      string    `json:"text" db:"text"`
	Type      string    `json:"type" db:"type"`
	Subtype   string    `json:"subtype" db:"subtype"`
	Timestamp string    `json:"ts" db:"timestamp"`
	Date      time.Time `db:"date"`
	Filename  string    `db:"filename"`
	// Thread information
	ThreadTS       string `json:"thread_ts" db:"thread_ts"`
	ParentUserID   string `json:"parent_user_id" db:"parent_user_id"`
	ReplyCount     int    `json:"reply_count" db:"reply_count"`
	ReplyUsersCount int   `json:"reply_users_count" db:"reply_users_count"`
	LatestReply    string `json:"latest_reply" db:"latest_reply"`
	// User information joined from users table
	UserName     string `db:"user_name"`
	UserRealName string `db:"user_real_name"`
}

// SearchResult represents a search result with context
type SearchResult struct {
	Message
	Rank     float64 `db:"rank"`
	Snippet  string  `db:"snippet"`
	Filename string  `db:"filename"`
}