package gmail

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/webbben/valet-de-chambre/internal/debug"
	emailcache "github.com/webbben/valet-de-chambre/internal/email_cache"
	"github.com/webbben/valet-de-chambre/internal/openai"
	t "github.com/webbben/valet-de-chambre/internal/types"
	"google.golang.org/api/gmail/v1"
)

func GetEmails(srv *gmail.Service, openAIKey string, config t.Config) []t.Email {
	debug.Println("getting emails...")
	emails := []t.Email{}
	r, err := srv.Users.Messages.List(config.GmailAddr).Do()
	if err != nil {
		log.Println("failed to list emails:", err)
		return emails
	}
	if len(r.Messages) == 0 {
		debug.Println("No emails found.")
		return emails
	}
	for _, msg := range r.Messages {
		if len(emails) >= config.EmailBatchLimit {
			break
		}
		if _, isCached := emailcache.IsCached(msg.Id); isCached {
			continue
		}
		email, err, add := processEmail(srv, msg.Id, config.GmailAddr)
		if err != nil {
			debug.Println("failed to process email:", err)
			continue
		}
		if isEmailTooOld(email, config) {
			debug.Println("email too old:", email.Date, email.From)
			emailcache.AddToCache(email, emailcache.IGNORE)
			break
		}
		if !add || isJunk(email, openAIKey) {
			emailcache.AddToCache(email, emailcache.IGNORE)
			continue
		}
		emails = append(emails, email)
	}
	debug.Println("... done!")
	return emails
}

// determines if the given email is junk or unwanted
func isJunk(email t.Email, openAIKey string) bool {
	if len(email.Body) == 0 {
		debug.Println("empty email?", email.From)
		return true
	}
	if len(email.Body) > 1500 {
		debug.Println("email too long:", len(email.Body), email.From)
		return true
	}
	if openai.IsEmailSpam(openAIKey, email.Body) {
		debug.Println("spam email:", email.From, email.Snippet)
		return true
	}
	return false
}

func processEmail(srv *gmail.Service, messageID string, emailAddr string) (t.Email, error, bool) {
	msg, err := srv.Users.Messages.Get(emailAddr, messageID).Do()
	if err != nil {
		return t.Email{}, err, false
	}
	// basic info
	email := t.Email{
		ID:       messageID,
		Snippet:  msg.Snippet,
		Date:     convInternalDateToTime(msg.InternalDate),
		ThreadID: msg.ThreadId,
	}
	// get headers
	for _, header := range msg.Payload.Headers {
		if header.Name == "From" {
			email.From = extractEmailAddr(header.Value)
		}
		if header.Name == "Subject" {
			email.Subject = header.Value
		}
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
		return email, nil, false
	}
	decoded, err := base64.URLEncoding.DecodeString(body)
	if err != nil {
		return email, err, false
	}
	email.Body = string(decoded)

	return email, nil, true
}

func SendReply(srv *gmail.Service, userID string, replyToEmail t.Email, replyBody string) error {
	replyMessage, err := createReply(replyToEmail, userID, replyBody)
	if err != nil {
		return err
	}
	_, err = srv.Users.Messages.Send(userID, replyMessage).Do()
	return err
}

func createReply(replyToEmail t.Email, userID string, replyBody string) (*gmail.Message, error) {
	if replyToEmail.ID == "" || replyToEmail.From == "" || userID == "" {
		return nil, errors.New("failed to create reply; missing required email properties")
	}
	if replyToEmail.ThreadID == "" {
		debug.Println("no thread ID present?")
	}
	// make the headers
	replySubject := "Re: " + replyToEmail.Subject
	headers := make(map[string]string)
	headers["From"] = userID
	headers["To"] = replyToEmail.From
	headers["Subject"] = replySubject
	headers["In-Reply-To"] = replyToEmail.ID
	headers["References"] = replyToEmail.ID
	headers["Date"] = time.Now().Format(time.RFC1123Z)
	var messageContent string
	for k, v := range headers {
		messageContent += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	messageContent += "\r\n" + replyBody

	// encode the email message
	encodedEmail := base64.URLEncoding.EncodeToString([]byte(messageContent))
	gMessage := &gmail.Message{
		Raw:      encodedEmail,
		ThreadId: replyToEmail.ThreadID,
	}
	return gMessage, nil
}
