package gmail

import (
	"fmt"
	"regexp"
	"time"

	t "github.com/webbben/valet-de-chambre/internal/types"
)

// To and From headers in Gmail may be formatted as "First Last <email.addr@gmail.com>"
// this extracts just the email address portion
func extractEmailAddr(emailAddrHeader string) string {
	// regex pattern for email addresses
	pattern := `(?i)([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(emailAddrHeader)
	if len(matches) > 0 {
		return matches[0]
	}
	// ... no email address found?
	return ""
}

func convInternalDateToTime(internalDate int64) time.Time {
	fmt.Println(internalDate)
	seconds := internalDate / 1000
	nanoseconds := (internalDate % 1000) * int64(time.Millisecond)
	return time.Unix(seconds, nanoseconds)
}

func isEmailTooOld(email t.Email, config t.Config) bool {
	if config.LookbackDays == 0 {
		return false
	}
	// email date is unset or at the "zero" value, so invalid to compare
	if email.Date.Equal(time.Time{}) {
		return false
	}
	return email.Date.Before(time.Now().Add((-24 * time.Hour) * time.Duration(config.LookbackDays)))
}
