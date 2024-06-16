package gmail

import (
	"encoding/base64"
	"regexp"
	"strings"
	"time"

	"github.com/webbben/mail-assistant/internal/config"
	t "github.com/webbben/mail-assistant/internal/types"
)

// To and From headers in Gmail may be formatted as "First Last <email.addr@gmail.com>"
// this extracts just the email address portion
func extractEmailAndName(emailAddrHeader string) (string, string) {
	// see if there's a name
	name := ""
	pieces := strings.Split(emailAddrHeader, "<")
	if len(pieces) > 1 {
		name = strings.TrimSpace(pieces[0])
	}
	// regex pattern for email addresses
	pattern := `(?i)([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(emailAddrHeader)
	if len(matches) > 0 {
		return matches[0], name
	}
	// ... no email address found?
	return "", ""
}

func convInternalDateToTime(internalDate int64) time.Time {
	seconds := internalDate / 1000
	nanoseconds := (internalDate % 1000) * int64(time.Millisecond)
	return time.Unix(seconds, nanoseconds)
}

func isEmailTooOld(email t.Email, config config.Config) bool {
	if config.LookbackDays == 0 {
		return false
	}
	// email date is unset or at the "zero" value, so invalid to compare
	if email.Date.Equal(time.Time{}) {
		return false
	}
	return email.Date.Before(time.Now().Add((-24 * time.Hour) * time.Duration(config.LookbackDays)))
}

func isEmailNoReply(email t.Email) bool {
	sender := strings.ToLower(email.From)
	return strings.Contains(sender, "noreply") || strings.Contains(sender, "no-reply")
}

func decodeRawMessage(raw string) (string, error) {
	bytes, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
