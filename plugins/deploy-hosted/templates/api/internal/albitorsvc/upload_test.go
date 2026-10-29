package albitorsvc

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateUploadSlotAndUpload(t *testing.T) {
	var parts []string
	var fileType, fileBody string
	store := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Errorf("content type: %v", err)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err != nil {
				break
			}
			b, _ := io.ReadAll(p)
			parts = append(parts, p.FormName())
			if p.FormName() == "file" {
				fileType, fileBody = p.Header.Get("Content-Type"), string(b)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer store.Close()

	var gotReq string
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotReq = r.URL.Path + " " + string(b)
		_, _ = w.Write([]byte(`{"UploadHandle":"01HZ","Url":"` + store.URL + `/bucket","Fields":{"key":"apps/x/in/01HZ","tagging":"<Tagging/>","policy":"p"},"ExpiresAt":"2026-10-29T10:05:00Z"}`))
	}))
	defer gw.Close()

	c := testClient(t, gw, nil)
	c.http = store.Client() // trusts the TLS test server; the gateway above is plain loopback
	slot, err := c.CreateUploadSlot(context.Background(), "transcribe", "audio/mpeg", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if gotReq != `/v1/transcribe/CreateUploadSlot {"ContentType":"audio/mpeg","MaxBytes":1024}` {
		t.Errorf("slot request = %q", gotReq)
	}
	if slot.Handle != "01HZ" || !slot.ExpiresAt.Equal(time.Date(2026, 10, 29, 10, 5, 0, 0, time.UTC)) {
		t.Errorf("slot = %+v", slot)
	}
	if err := c.Upload(context.Background(), slot, []byte("audio-bytes")); err != nil {
		t.Fatal(err)
	}
	if strings.Join(parts, ",") != "key,policy,tagging,file" {
		t.Errorf("parts = %v, want the fields then the file last", parts)
	}
	if fileType != "audio/mpeg" || fileBody != "audio-bytes" {
		t.Errorf("file part = %q %q", fileType, fileBody)
	}
}

func TestUploadRefusesWhatThePolicyWouldAndHidesTheStoreBody(t *testing.T) {
	store := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<Error><Key>apps/other/in/secret</Key></Error>"))
	}))
	defer store.Close()
	c := testClient(t, store, nil)
	slot := Slot{URL: store.URL, Fields: map[string]string{"key": "k"}, MaxBytes: 4, ContentType: "audio/mpeg"}

	if err := c.Upload(context.Background(), slot, []byte("12345")); err == nil {
		t.Error("an upload over the slot's MaxBytes was sent")
	}
	if err := c.Upload(context.Background(), slot, nil); err == nil {
		t.Error("an empty upload was sent")
	}
	err := c.Upload(context.Background(), slot, []byte("1234"))
	if err == nil || strings.Contains(err.Error(), "secret") || !strings.Contains(err.Error(), "403") {
		t.Errorf("err = %v, want the status only", err)
	}
	slot.URL = "http://insecure.example.com/"
	if err := c.Upload(context.Background(), slot, []byte("1")); err == nil {
		t.Error("an upload to a plain-http URL was sent")
	}
}
