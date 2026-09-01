package files

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestSendToChatSuccess(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1,"chat":{"id":"c1","type":"private"},"from":{"login":"u1"},"text":"file"}}`),
		},
	}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, doer)

	svc := NewService(client)
	msg, err := svc.SendToChat(context.Background(), "c1", "f.txt", "text/plain", []byte("hello"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil || msg.ID != 1 {
		t.Fatalf("expected message m1")
	}
	if len(doer.Requests) != 1 {
		t.Fatalf("expected one request")
	}
	req := doer.Requests[0]
	if req.Method != http.MethodPost {
		t.Fatalf("expected POST, got %s", req.Method)
	}
	if req.URL.Path != "/bot/v1/messages/sendFile" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}
	if ct := req.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/form-data") {
		t.Fatalf("expected multipart content type, got %s", ct)
	}
}

func TestSendToLoginInvalidResponse(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":false}`),
		},
	})

	svc := NewService(client)
	_, err := svc.SendToLogin(context.Background(), "login1", "f.txt", "text/plain", []byte("hello"), nil)
	if !errors.Is(err, ymerrors.ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestSendToChatBadJSON(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":`),
		},
	})

	svc := NewService(client)
	_, err := svc.SendToChat(context.Background(), "c1", "f.txt", "text/plain", []byte("hello"), nil)
	if !errors.Is(err, ymerrors.ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{},
	}
}

// The Bot API sendFile method documents no caption parameter, so the SDK must
// keep one off the wire. An unknown multipart part is discarded server-side
// without an error, which makes a caption look delivered when it never was.
func TestSendToChatOmitsUndocumentedCaption(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1,"chat":{"id":"c1","type":"private"},"from":{"login":"u1"},"text":"file"}}`),
		},
	}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, doer)

	svc := NewService(client)
	opts := &SendFileOptions{Caption: "must not reach the wire"}
	if _, err := svc.SendToChat(context.Background(), "c1", "f.txt", "text/plain", []byte("hello"), opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fields, _ := decodeMultipart(t, doer.Requests[0])
	if got, ok := fields["caption"]; ok {
		t.Fatalf("caption must not be sent: sendFile has no such parameter, got %q", got)
	}
	if fields["chat_id"] != "c1" {
		t.Fatalf("expected chat_id c1, got %q", fields["chat_id"])
	}
}

// MimeType is the other half of SendFileOptions; this guards it against the
// caption removal touching the shared multipart builder.
func TestSendToChatAppliesMimeTypeOverride(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1,"chat":{"id":"c1","type":"private"},"from":{"login":"u1"},"text":"file"}}`),
		},
	}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, doer)

	svc := NewService(client)
	opts := &SendFileOptions{MimeType: "application/pdf"}
	if _, err := svc.SendToChat(context.Background(), "c1", "f.pdf", "text/plain", []byte("hello"), opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, fileTypes := decodeMultipart(t, doer.Requests[0])
	if fileTypes["document"] != "application/pdf" {
		t.Fatalf("expected MimeType to override the document part type, got %q", fileTypes["document"])
	}
}

// decodeMultipart replays a recorded request body and splits it into plain form
// fields and the Content-Type of each file part.
func decodeMultipart(t *testing.T, req *http.Request) (map[string]string, map[string]string) {
	t.Helper()

	if req.GetBody == nil {
		t.Fatalf("recorded request has no replayable body")
	}
	body, err := req.GetBody()
	if err != nil {
		t.Fatalf("replay body: %v", err)
	}
	defer body.Close()

	_, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse media type: %v", err)
	}

	fields, fileTypes := map[string]string{}, map[string]string{}
	reader := multipart.NewReader(body, params["boundary"])
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read next part: %v", err)
		}
		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read part %q: %v", part.FormName(), err)
		}
		if part.FileName() != "" {
			fileTypes[part.FormName()] = part.Header.Get("Content-Type")

			continue
		}
		fields[part.FormName()] = string(data)
	}

	return fields, fileTypes
}
