package gmail

import (
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"google.golang.org/api/gmail/v1"
)

type Email struct {
	From    string
	Subject string
	Body    string
	Snippet string
	Date    time.Time
}

func GetEmails(srv *gmail.Service, emailAddr string, limit int) []Email {
	fmt.Println("getting emails...")
	emails := []Email{}
	r, err := srv.Users.Messages.List(emailAddr).Do()
	if err != nil {
		log.Println("failed to list emails:", err)
		return emails
	}
	if len(r.Messages) == 0 {
		log.Println("No emails found.")
		return emails
	}
	for _, msg := range r.Messages {
		if len(emails) >= limit {
			break
		}
		email, err, add := processEmail(srv, msg.Id, emailAddr)
		if err != nil {
			log.Println("failed to process email:", err)
		}
		if !add || isJunk(email) {
			continue
		}
		emails = append(emails, email)
	}
	fmt.Println("... done!")
	return emails
}

func isJunk(email Email) bool {
	if len(email.Body) == 0 {
		log.Println("empty email?", email.From)
		return true
	}
	if len(email.Body) > 20000 {
		log.Println("email too long.", email.From)
		return true
	}
	return false
}

func processEmail(srv *gmail.Service, messageID string, emailAddr string) (Email, error, bool) {
	msg, err := srv.Users.Messages.Get(emailAddr, messageID).Do()
	if err != nil {
		return Email{}, err, false
	}
	msgDate := time.Unix(msg.InternalDate, 0)

	// if the message more than a day old, ignore it
	if msgDate.Before(time.Now().Add(-24 * time.Hour)) {
		log.Println("email too old?")
		//return Email{}, nil, false
	}

	// get the email content
	body := ""
	for _, part := range msg.Payload.Parts {
		if part.MimeType == "text/plain" {
			body = part.Body.Data
			break
		}
	}
	if body == "" {
		log.Println("no plain text body found.")
		return Email{}, nil, false
	}
	decoded, err := base64.URLEncoding.DecodeString(body)
	if err != nil {
		return Email{}, err, false
	}
	email := Email{
		Body:    string(decoded),
		Snippet: msg.Snippet,
		Date:    msgDate,
	}

	// get headers
	for _, header := range msg.Payload.Headers {
		if header.Name == "From" {
			email.From = header.Value
		}
		if header.Name == "Subject" {
			email.Subject = header.Value
		}
	}
	return email, nil, true
}
