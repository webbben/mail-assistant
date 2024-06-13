package emailparse

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
)

func init() {
	message.CharsetReader = charset.Reader
}

func ParseEmail(rawEmail string) (string, map[string][]string, error) {
	r := strings.NewReader(rawEmail)
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return "", nil, err
	}

	headers, err := extractAllHeaders(msg.Header)
	if err != nil {
		log.Println("failed to extract headers:", err)
		return "", nil, err
	}

	// Get the plain text content from the email body
	contentType := msg.Header.Get("Content-Type")
	encoding := msg.Header.Get("Content-Transfer-Encoding")
	plainText, err := extractPlainText(msg.Body, contentType, encoding)
	if err != nil {
		return "", nil, err
	}
	return plainText, headers, nil
}

func extractAllHeaders(header mail.Header) (map[string][]string, error) {
	headers := make(map[string][]string)
	for k, v := range header {
		for _, val := range v {
			decodedValue, err := decodeHeader(val)
			if err != nil {
				return nil, err
			}
			headers[k] = append(headers[k], decodedValue)
		}
	}
	// for some reason Message-ID may not be found in the above loop, and we need to get it manually...?
	if _, exists := headers["Message-ID"]; !exists {
		headers["Message-ID"] = []string{header.Get("Message-ID")}
	}
	return headers, nil
}

func decodeHeader(header string) (string, error) {
	dec := new(mime.WordDecoder)
	return dec.DecodeHeader(header)
}

// extracts plain text from the email body. if multipart, will recursively look until text/plain or text/html content found.
func extractPlainText(body io.Reader, contentType string, encoding string) (string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(mediaType, "text/plain") {
		return parsePlainText(body, encoding)
	}
	if strings.HasPrefix(mediaType, "text/html") {
		return parseHTML(body), nil
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		if params["boundary"] == "" {
			return "", errors.New("multipart: no boundary in params")
		}
		mr := multipart.NewReader(body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			contentType := part.Header.Get("Content-Type")
			encoding := part.Header.Get("Content-Transfer-Encoding")
			plainText, err := extractPlainText(part, contentType, encoding)
			if err == nil && plainText != "" {
				return plainText, nil
			}
		}
	}

	return "", fmt.Errorf("no plain text content found: " + mediaType)
}

func parsePlainText(body io.Reader, encoding string) (string, error) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(body)
	var err error
	s := strings.TrimSpace(buf.String())
	if encoding == "base64" {
		s, err = decodeBase64String(s)
		if err != nil {
			return "", err
		}
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s, nil
}

func parseHTML(body io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(body)
	plainText := html.UnescapeString(stripHTMLTags(buf.String()))
	return strings.TrimSpace(plainText)
}

// finds plain text inside an html string
func stripHTMLTags(htmlStr string) string {
	var result []rune
	inTag := false
	for _, r := range htmlStr {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result = append(result, r)
			}
		}
	}
	return string(result)
}

func decodeBase64String(s string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(decodedBytes)), nil
}
