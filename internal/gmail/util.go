package gmail

import (
	"regexp"
	"time"
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
	seconds := internalDate / 1000
	nanoseconds := (internalDate % 1000) * int64(time.Millisecond)
	return time.Unix(seconds, nanoseconds)
}
