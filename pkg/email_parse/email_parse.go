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
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// maps charset names to their respective encodings.
var decoderMap = map[string]encoding.Encoding{
	"utf-8":       encoding.Nop, // No transformation needed for UTF-8
	"iso-2022-jp": japanese.ISO2022JP,
	"shift_jis":   japanese.ShiftJIS,
	"euc-jp":      japanese.EUCJP,
	"iso-8859-1":  charmap.ISO8859_1,
}

func init() {
	message.CharsetReader = charset.Reader
}

func ParseEmail(rawEmail string) (string, map[string][]string, error) {
	r := strings.NewReader(rawEmail)
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return "", nil, errors.Join(errors.New("failed to read message"), err)
	}

	headers, err := extractAllHeaders(msg.Header)
	if err != nil {
		return "", nil, errors.Join(errors.New("failed to extract headers"), err)
	}

	// Get the plain text content from the email body
	contentType := msg.Header.Get("Content-Type")
	encoding := msg.Header.Get("Content-Transfer-Encoding")
	plainText, err := extractPlainText(msg.Body, contentType, encoding)
	if err != nil {
		return "", nil, errors.Join(errors.New("failed to extract text"), err)
	}
	return plainText, headers, nil
}

func extractAllHeaders(header mail.Header) (map[string][]string, error) {
	headers := make(map[string][]string)
	for k, v := range header {
		for _, val := range v {
			decodedValue, err := decodeHeader(val)
			if err != nil {
				log.Println("failed to decode header: "+k, err)
				//return nil, err
				decodedValue = val
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

func _decodeHeader(header string) (string, error) {
	dec := new(mime.WordDecoder)
	return dec.DecodeHeader(header)
}

func decodeHeader(header string) (string, error) {
	dec := new(mime.WordDecoder)
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		data, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		decoded, err := decodeToUTF8(string(data), charset)
		if err != nil {
			return nil, err
		}
		return strings.NewReader(decoded), nil
	}
	return dec.DecodeHeader(header)
}

// extracts plain text from the email body. if multipart, will recursively look until text/plain or text/html content found.
func extractPlainText(body io.Reader, contentType string, encoding string) (string, error) {
	// use a default content type header like this, if it's missing for some reason
	if contentType == "" {
		contentType = "text/plain; charset=\"utf-8\""
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", errors.Join(errors.New("failed to parse media type"), err)
	}

	if strings.HasPrefix(mediaType, "text/plain") {
		return parsePlainText(body, encoding, params["charset"])
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

func parsePlainText(body io.Reader, encoding string, charset string) (string, error) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(body)
	var err error
	s := strings.TrimSpace(buf.String())
	if encoding == "base64" {
		s, err = decodeBase64String(s)
		if err != nil {
			return "", err
		}
	} else if encoding == "quoted-printable" {
		s, err = decodeQuotedPrintableString(s)
		if err != nil {
			return "", err
		}
	}
	charset = strings.ToLower(charset)
	if charset != "" && charset != "utf-8" && charset != "us-ascii" {
		s, err = decodeToUTF8(s, charset)
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

func decodeQuotedPrintableString(s string) (string, error) {
	decoder := quotedprintable.NewReader(strings.NewReader(s))
	bytes, err := io.ReadAll(decoder)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func decodeToUTF8(s, charset string) (string, error) {
	charset = strings.ToLower(charset)
	if charset == "utf-8" {
		return s, nil
	}
	enc, exists := decoderMap[charset]
	if !exists {
		return "", errors.New("unsupported charset: " + charset)
	}
	decoder := enc.NewDecoder()
	decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader([]byte(s)), decoder))
	if err != nil {
		return "", fmt.Errorf("failed to decode: %v", err)
	}

	return string(decoded), nil
}
