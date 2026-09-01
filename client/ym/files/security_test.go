package files

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/internal/testutil"
)

// The messages service strips CR and LF from filenames because multipart part
// headers are written verbatim. This low-level service builds its own
// Content-Disposition and needs the same protection — a filename is often the
// least trusted string in the request.
func TestSendStripsHeaderInjectionFromFilename(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1}}`),
	}}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	_, err := svc.SendToChat(context.Background(), "c1",
		"ok.txt\r\nX-Injected: yes\r\n\r\nsmuggled", "text/plain", []byte("payload"), nil)
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
}
