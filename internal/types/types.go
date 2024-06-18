package types

import (
	"fmt"
	"time"
)

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

func (e Email) String() string {
	from := e.From
	if e.SenderName != "" {
		from = fmt.Sprintf("%s (%s)", from, e.SenderName)
	}
	return fmt.Sprintf("From: %s\nSubject: %s\n\n%s", from, e.Subject, e.Body)
}
