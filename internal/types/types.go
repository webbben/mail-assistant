package types

import "time"

// a processed email
type Email struct {
	ID         string
	ThreadID   string
	From       string
	SenderName string
	Subject    string
	Body       string
	Snippet    string
	Date       time.Time
}
