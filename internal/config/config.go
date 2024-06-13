package config

import (
	"encoding/json"
	"os"
)

// configuration for this application
type Config struct {
	UserName        string `json:"user_name"`         // name of the human using this application
	PersonalityID   string `json:"personality_id"`    // id of the personality file for the AI to use
	GmailAddr       string `json:"gmail_address"`     // gmail address to use
	InboxCheckFreq  int    `json:"inbox_check_freq"`  // frequency in minutes in which the gmail inbox is checked for mail
	EmailBatchLimit int    `json:"email_batch_limit"` // limit to the number of emails that will be processed in a single batch
	LookbackDays    int    `json:"lookback_days"`     // number of days to look back in the inbox (0 = no limit)
	Debug           bool   `json:"debug"`             // if enabled, debug statements will be printed to the console
}

func LoadConfig() (Config, error) {
	bytes, err := os.ReadFile("config.json")
	if err != nil {
		return Config{}, err
	}
	var c Config
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return Config{}, err
	}
	return c, nil
}
