package gmail

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/webbben/mail-assistant/internal/config"
	"github.com/webbben/mail-assistant/internal/debug"
	emailcache "github.com/webbben/mail-assistant/internal/email_cache"
	"github.com/webbben/mail-assistant/internal/llama"
	t "github.com/webbben/mail-assistant/internal/types"
	emailparse "github.com/webbben/mail-assistant/pkg/email_parse"
	"google.golang.org/api/gmail/v1"
)

const (
	SPAM     = "SPAM"
	BAD_FORM = "BAD_FORM"
	NOREPLY  = "NOREPLY"
	OLD      = "OLD"
)

func ListMessages(srv *gmail.Service, gmailAddr string) ([]*gmail.Message, error) {
	r, err := srv.Users.Messages.List(gmailAddr).Do()
	if err != nil {
		return nil, err
	}
	return r.Messages, nil
}

func GetMessage(srv *gmail.Service, gmailAddr string, messageID string) (*gmail.Message, error) {
	return srv.Users.Messages.Get(gmailAddr, messageID).Format("raw").Do()
}

func GetRaw(srv *gmail.Service, gmailAddr string, messageID string) (string, error) {
	msg, err := srv.Users.Messages.Get(gmailAddr, messageID).Format("raw").Do()
	if err != nil {
		return "", err
	}
	return decodeRawMessage(msg.Raw)
}

func GetEmails(srv *gmail.Service, ollamaClient *api.Client, config config.Config) []t.Email {
	debug.Println("getting emails...")
	emails := []t.Email{}
	list, err := ListMessages(srv, config.GmailAddr)
	if err != nil {
		log.Println("failed to list emails:", err)
		return emails
	}
	if len(list) == 0 {
		debug.Println("No emails found.")
		return emails
	}
	for _, msg := range list {
		if len(emails) >= config.EmailBatchLimit {
			break
		}
		if _, isCached := emailcache.IsCached(msg.Id); isCached {
			continue
		}
		email, err := ProcessEmail(srv, msg.Id, config.GmailAddr)
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
	if len(email.Body) > 3000 {
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

func ProcessEmail(srv *gmail.Service, messageID string, emailAddr string) (t.Email, error) {
	msg, err := GetMessage(srv, emailAddr, messageID)
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
	// get the email content
	raw, err := decodeRawMessage(msg.Raw)
	if err != nil {
		return email, err
	}
	body, headers, err := emailparse.ParseEmail(raw)
	if err != nil {
		return email, err
	}
	email.From = headers["From"][0]
	email.Subject = headers["Subject"][0]
	email.Body = body
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
