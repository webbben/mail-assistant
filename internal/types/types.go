package types

import "time"

// configuration for this application
type Config struct {
	UserName        string `json:"user_name"`         // name of the human using this application
	PromptID        string `json:"prompt_id"`         // id of the prompt text file to use when starting AI interaction
	GmailAddr       string `json:"gmail_address"`     // gmail address to use
	AIName          string `json:"ai_name"`           // Name of the AI email assistant
	InboxCheckFreq  int    `json:"inbox_check_freq"`  // frequency in minutes in which the gmail inbox is checked for mail
	EmailBatchLimit int    `json:"email_batch_limit"` // limit to the number of emails that will be processed in a single batch
	LookbackDays    int    `json:"lookback_days"`     // number of days to look back in the inbox (0 = no limit)
}

// a processed email
type Email struct {
	ID       string
	ThreadID string
	From     string
	Subject  string
	Body     string
	Snippet  string
	Date     time.Time
}
