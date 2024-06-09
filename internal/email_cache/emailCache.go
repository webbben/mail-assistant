package emailcache

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	t "github.com/webbben/valet-de-chambre/internal/types"
)

const (
	IGNORE = "IGNORE"
	REPLY  = "REPLY"
)

var cache map[string]EmailCacheDatum = make(map[string]EmailCacheDatum)

type EmailCacheDatum struct {
	MessageID  string
	Date       time.Time // date the email was created
	From       string    // who is the email from
	Action     string    // how was this email dealt with
	Categories string    // different categories attributed to this email (e.g. spam, noreply, etc)
}

func (datum EmailCacheDatum) String() string {
	return fmt.Sprintf("%s %s %v %s %s", datum.MessageID, datum.From, datum.Date.Unix(), datum.Action, datum.Categories)
}

func parseCacheLine(line string) (EmailCacheDatum, error) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return EmailCacheDatum{}, errors.New("cache line data corrupted or of incorrect format: " + line)
	}
	timestamp, err := strconv.Atoi(fields[2])
	if err != nil {
		return EmailCacheDatum{}, err
	}

	datum := EmailCacheDatum{
		MessageID: fields[0],
		From:      fields[1],
		Date:      time.Unix(int64(timestamp), 0),
		Action:    fields[3],
	}
	if len(fields) > 4 {
		datum.Categories = fields[4]
	}
	return datum, nil
}

// adds an email to the next batch of data to be cached
func AddToCache(email t.Email, action string, categories ...string) {
	cache[email.ID] = EmailCacheDatum{
		MessageID:  email.ID,
		From:       strings.ReplaceAll(email.From, " ", "_"), // just in case some spaces somehow snuck in
		Date:       email.Date,
		Action:     action,
		Categories: strings.Join(categories, ";"),
	}
}

// checks the in-memory cache for the given message ID, and also returns its cache data if found
func IsCached(messageID string) (EmailCacheDatum, bool) {
	datum, isCached := cache[messageID]
	return datum, isCached
}

// writes the current data in the in-memory cache to the disk, overwriting any previous data in the file
func WriteCacheToDisk() error {
	file, err := os.OpenFile("emailcache", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, datum := range cache {
		_, err := writer.WriteString(datum.String() + "\n")
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}

// loads the cache from its file on the disk into application memory to be accessible
func LoadCacheFromDisk() error {
	file, err := os.Open("emailcache")
	if err != nil {
		return err
	}
	defer file.Close()

	cache = make(map[string]EmailCacheDatum)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		datum, err := parseCacheLine(line)
		if err != nil {
			return err
		}
		cache[datum.MessageID] = datum
	}
	return nil
}

// removes entries in the cache that are too old and will no longer be processed
//
// lookbackDays should match the value in your configuration json
func RemoveOldEntries(lookbackDays int) {
	if lookbackDays == 0 {
		return
	}
	for id, datum := range cache {
		if datum.Date.Before(time.Now().Add(-24 * time.Hour * time.Duration(lookbackDays))) {
			delete(cache, id)
		}
	}
}
