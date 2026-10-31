package albitorsvc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"sort"
	"time"
)

// Slot is a one-shot place to put an input. It is a credential: it expires in
// minutes, so upload at once, and never log it, store it or return it to a
// browser. An operation's request names the input only by Handle.
type Slot struct {
	Handle    string
	URL       string
	Fields    map[string]string
	ExpiresAt time.Time
	// MaxBytes and ContentType are what the slot was asked for; the upload
	// policy refuses anything else.
	MaxBytes    int64
	ContentType string
}

type slotRequest struct {
	ContentType string `json:"ContentType"`
	MaxBytes    int64  `json:"MaxBytes"`
}

type slotResponse struct {
	UploadHandle string            `json:"UploadHandle"`
	URL          string            `json:"Url"`
	Fields       map[string]string `json:"Fields"`
	ExpiresAt    string            `json:"ExpiresAt"`
}

// CreateUploadSlot asks a service that takes inputs (transcribe, textract, ...)
// for a slot. contentType must be one the service accepts and maxBytes within its
// cap, or the gateway answers 400 bad_request.
func (c *Client) CreateUploadSlot(ctx context.Context, service, contentType string, maxBytes int64) (Slot, error) {
	var r slotResponse
	if err := c.Call(ctx, service, "CreateUploadSlot", slotRequest{ContentType: contentType, MaxBytes: maxBytes}, &r); err != nil {
		return Slot{}, err
	}
	exp, err := time.Parse(time.RFC3339, r.ExpiresAt)
	if err != nil || r.UploadHandle == "" || r.URL == "" {
		return Slot{}, errors.New("albitorsvc: malformed upload slot")
	}
	return Slot{Handle: r.UploadHandle, URL: r.URL, Fields: r.Fields, ExpiresAt: exp, MaxBytes: maxBytes, ContentType: contentType}, nil
}

// Upload sends data to the slot as the multipart form a presigned POST expects:
// its fields first, the file last. It is not signed with the app's credentials;
// the slot is the credential. A failure carries the status only, never the
// storage service's body, which can name the key.
func (c *Client) Upload(ctx context.Context, s Slot, data []byte) error {
	if len(data) == 0 || int64(len(data)) > s.MaxBytes {
		return fmt.Errorf("albitorsvc: upload is %d bytes, the slot allows 1 to %d", len(data), s.MaxBytes)
	}
	u, err := url.Parse(s.URL)
	if err != nil || u.Scheme != "https" {
		return errors.New("albitorsvc: the upload URL must be https")
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	names := make([]string, 0, len(s.Fields))
	for k := range s.Fields {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		if err := mw.WriteField(k, s.Fields[k]); err != nil {
			return err
		}
	}
	// The policy fixes the content type, so the file part carries the same one.
	fw, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="upload"`},
		"Content-Type":        {s.ContentType},
	})
	if err != nil {
		return err
	}
	if _, err := fw.Write(data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("albitorsvc: the upload failed to send")
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxErrorBody))
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("albitorsvc: upload refused with status %d", res.StatusCode)
	}
	return nil
}
