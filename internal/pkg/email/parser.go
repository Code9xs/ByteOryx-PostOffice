package email

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

type ParsedMessage struct {
	Subject     string
	From        string
	To          []string
	CC          []string
	MessageID   string
	InReplyTo   string
	Date        string
	BodyText    string
	BodyHTML    string
	Attachments []ParsedAttachment
}

type ParsedAttachment struct {
	Filename    string
	ContentType string
	Size        int
	Data        []byte
}

func Parse(raw []byte) (*ParsedMessage, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	parsed := &ParsedMessage{
		Subject:   decodeHeader(msg.Header.Get("Subject")),
		From:      msg.Header.Get("From"),
		MessageID: msg.Header.Get("Message-ID"),
		InReplyTo: msg.Header.Get("In-Reply-To"),
		Date:      msg.Header.Get("Date"),
	}

	if to := msg.Header.Get("To"); to != "" {
		parsed.To = splitAddresses(to)
	}
	if cc := msg.Header.Get("Cc"); cc != "" {
		parsed.CC = splitAddresses(cc)
	}

	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		body, _ := io.ReadAll(msg.Body)
		parsed.BodyText = string(body)
		return parsed, nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary != "" {
			parseMultipart(msg.Body, boundary, parsed)
		}
	} else {
		body, _ := io.ReadAll(msg.Body)
		if strings.Contains(mediaType, "html") {
			parsed.BodyHTML = string(body)
		} else {
			parsed.BodyText = string(body)
		}
	}

	return parsed, nil
}

func parseMultipart(r io.Reader, boundary string, parsed *ParsedMessage) {
	mr := multipart.NewReader(r, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}

		ct := part.Header.Get("Content-Type")
		mediaType, params, _ := mime.ParseMediaType(ct)
		disposition := part.Header.Get("Content-Disposition")

		data, _ := io.ReadAll(part)

		if strings.Contains(disposition, "attachment") || strings.Contains(disposition, "inline") {
			filename := part.FileName()
			if filename == "" {
				filename = "unnamed"
			}
			parsed.Attachments = append(parsed.Attachments, ParsedAttachment{
				Filename:    filename,
				ContentType: mediaType,
				Size:        len(data),
				Data:        data,
			})
		} else if strings.HasPrefix(mediaType, "multipart/") {
			subBoundary := params["boundary"]
			if subBoundary != "" {
				parseMultipart(bytes.NewReader(data), subBoundary, parsed)
			}
		} else if strings.Contains(mediaType, "html") {
			parsed.BodyHTML = string(data)
		} else if strings.Contains(mediaType, "text") {
			parsed.BodyText = string(data)
		}
	}
}

func splitAddresses(s string) []string {
	addrs, err := mail.ParseAddressList(s)
	if err != nil {
		return []string{s}
	}
	result := make([]string, len(addrs))
	for i, a := range addrs {
		result[i] = a.Address
	}
	return result
}

func decodeHeader(s string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}
