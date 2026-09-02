package messages

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

// Multipart part headers are written verbatim, so a newline in a filename would
// end the Content-Disposition header and let the rest be read as new MIME
// headers or body content.
func TestSendFileStripsHeaderInjectionFromFilename(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`)},
	}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	_, err := svc.SendFile(context.Background(), &SendFileRequest{
		ChatID:   ym.Ptr(ym.ChatID("c1")),
		Document: strings.NewReader("payload"),
		Filename: "ok.txt\r\nX-Injected: yes\r\n\r\nsmuggled",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	body, err := io.ReadAll(doer.Requests[0].Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if bytes.Contains(body, []byte("X-Injected: yes\r\n")) {
		t.Fatalf("filename smuggled a MIME header into the part:\n%s", body)
	}
	if !bytes.Contains(body, []byte(`filename="ok.txtX-Injected: yessmuggled"`)) {
		t.Fatalf("expected the newlines to be stripped, got:\n%s", body)
	}
}

// A JSON body is an error envelope, not file bytes. Returning the reader after
// decoding it would hand the caller a drained, closed stream that looks like an
// empty file.
func TestGetFileNeverReturnsADrainedBody(t *testing.T) {
	resp := testutil.NewResponse(http.StatusOK, `{"ok":true}`)
	resp.Header.Set("Content-Type", "application/json")
	doer := &testutil.FakeDoer{Responses: []*http.Response{resp}}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	body, meta, err := svc.GetFile(context.Background(), "f1")
	if body != nil {
		t.Fatal("a JSON envelope must not be returned as file content")
	}
	if meta != nil {
		t.Fatalf("expected no metadata, got %#v", meta)
	}
	if !errors.Is(err, ymerrors.ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestSendRejectsOverLongText(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`)},
	}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	_, err := svc.SendToChat(context.Background(), "c1", strings.Repeat("a", ym.MaxTextLength+1), nil)

	var limitErr *ym.LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected a LimitError, got %v", err)
	}
	if doer.CallCount() != 0 {
		t.Fatalf("an over-long message must not reach the network, got %d calls", doer.CallCount())
	}
}

func TestSendRejectsOversizedKeyboard(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`)},
	}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	_, err := svc.SendToChat(context.Background(), "c1", "hi", &SendMessageOptions{
		ActionButtons: &ym.ActionButtons{Buttons: make([]ym.ActionButton, ym.MaxActionButtons+1)},
	})
	if err == nil {
		t.Fatal("expected the button limit to be enforced")
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}
