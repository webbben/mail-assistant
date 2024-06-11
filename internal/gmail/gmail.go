package gmail

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/webbben/mail-assistant/internal/debug"
	emailcache "github.com/webbben/mail-assistant/internal/email_cache"
	"github.com/webbben/mail-assistant/internal/llama"
	t "github.com/webbben/mail-assistant/internal/types"
	"google.golang.org/api/gmail/v1"
)

const (
	SPAM     = "SPAM"
	BAD_FORM = "BAD_FORM"
	NOREPLY  = "NOREPLY"
	OLD      = "OLD"
)

func GetEmails(srv *gmail.Service, ollamaClient *api.Client, config t.Config) []t.Email {
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
		email, err := processEmail(srv, msg.Id, config.GmailAddr)
		if err != nil {
			debug.Println("failed to process email:", err)
			continue
		}
		if isEmailTooOld(email, config) {
			debug.Println("email too old:", email.Date, email.From)
			emailcache.AddToCache(email, emailcache.IGNORE, OLD)
			break
		}
		if junk, reason := isJunk(email, ollamaClient); junk {
			emailcache.AddToCache(email, emailcache.IGNORE, reason)
			continue
		}
		emails = append(emails, email)
	}
	debug.Println("... done!")
	return emails
}

// determines if the given email is junk or unwanted, and if so, gives a category for why it is unwanted
func isJunk(email t.Email, ollamaClient *api.Client) (bool, string) {
	if len(email.Body) == 0 {
		debug.Println("empty email:", email.From)
		return true, BAD_FORM
	}
	if len(email.Body) > 1500 {
		debug.Println("email too long:", len(email.Body), email.From)
		return true, BAD_FORM
	}
	if isEmailNoReply(email) {
		debug.Println("no reply email:", email.From)
		return true, NOREPLY
	}
	if llama.IsEmailSpam(ollamaClient, email.Body) {
		debug.Println("spam email:", email.From, email.Snippet)
		return true, SPAM
	}
	return false, ""
}

func processEmail(srv *gmail.Service, messageID string, emailAddr string) (t.Email, error) {
	msg, err := srv.Users.Messages.Get(emailAddr, messageID).Do()
	if err != nil {
		return t.Email{}, err
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
			addr, name := extractEmailAndName(header.Value)
			email.From = addr
			email.SenderName = name
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
	decoded, err := base64.URLEncoding.DecodeString(body)
	if err != nil {
		return email, err
	}
	email.Body = string(decoded)

	return email, nil
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
