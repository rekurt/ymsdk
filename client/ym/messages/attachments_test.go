package messages

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"

	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestSendFileValidation(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.SendFile(context.Background(), &SendFileRequest{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestSendFileSuccess(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`),
		},
	})
	svc := NewService(client)
	msg, err := svc.SendFile(context.Background(), &SendFileRequest{
		ChatID:   ptrChat("c1"),
		Document: bytes.NewBufferString("data"),
		Filename: "f.txt",
	})
	if err != nil || msg == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteMessageAPIError(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":false,"description":"denied"}`),
		},
	})
	svc := NewService(client)
	err := svc.Delete(context.Background(), &DeleteMessageRequest{
		ChatID:    ptrChat("c1"),
		MessageID: 1,
	})
	var apiErr *ymerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected api error")
	}
}

func TestGetFileJSONError(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{})
	svc := NewService(client)
	resp := testutil.NewResponse(http.StatusOK, `{"ok":false,"description":"not found"}`)
	resp.Header.Set("Content-Type", "application/json")
	client.HTTPDoer().(*testutil.FakeDoer).Responses = []*http.Response{resp}

	_, _, err := svc.GetFile(context.Background(), "file1")
	var apiErr *ymerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected api error")
	}
}

func TestSendImageSuccess(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":42}`),
		},
	})
	svc := NewService(client)
	msg, err := svc.SendImage(context.Background(), &SendImageRequest{
		ChatID:   ptrChat("c1"),
		Image:    bytes.NewBufferString("png-data"),
		Filename: "photo.png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil || msg.ID != 42 {
		t.Fatalf("expected message with id=42, got %v", msg)
	}
}

func TestSendImageValidation(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.SendImage(context.Background(), &SendImageRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSendGallerySuccess(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":99}`),
		},
	})
	svc := NewService(client)
	msg, err := svc.SendGallery(context.Background(), &SendGalleryRequest{
		ChatID: ptrChat("c1"),
		Images: []FilePart{
			{Reader: bytes.NewBufferString("img1"), Filename: "a.jpg"},
			{Reader: bytes.NewBufferString("img2"), Filename: "b.jpg"},
		},
		Text: "gallery caption",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil || msg.ID != 99 {
		t.Fatalf("expected message with id=99, got %v", msg)
	}
}

func TestSendGalleryTooManyImages(t *testing.T) {
	svc := NewService(nil)
	images := make([]FilePart, 11)
	for i := range images {
		images[i] = FilePart{Reader: bytes.NewBufferString("x"), Filename: "x.jpg"}
	}
	_, err := svc.SendGallery(context.Background(), &SendGalleryRequest{
		ChatID: ptrChat("c1"),
		Images: images,
	})
	if err == nil {
		t.Fatal("expected error for too many images")
	}
}

func TestGetFileSuccess(t *testing.T) {
	resp := testutil.NewResponse(http.StatusOK, "file-content")
	resp.Header.Set("Content-Type", "application/octet-stream")
	resp.ContentLength = 12
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{resp},
	})
	svc := NewService(client)
	body, meta, err := svc.GetFile(context.Background(), "disk/abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()
	if meta.FileID != "disk/abc" {
		t.Fatalf("expected file_id=disk/abc, got %s", meta.FileID)
	}
	if meta.ContentType != "application/octet-stream" {
		t.Fatalf("unexpected content type: %s", meta.ContentType)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`normal.txt`, `normal.txt`},
		{`file"name.txt`, `file\"name.txt`},
		{`path\file.txt`, `path\\file.txt`},
		{`"evil\".txt`, `\"evil\\\".txt`},
		// Newlines would terminate the Content-Disposition header, so they are
		// removed rather than escaped.
		{"a\r\nb.txt", "ab.txt"},
		{"ok.txt\r\nX-Injected: yes", "ok.txtX-Injected: yes"},
	}
	for _, tc := range tests {
		got := sanitizeFilename(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func ptrChat(id ym.ChatID) *ym.ChatID { return &id }

// decodeDocumentPart returns the headers of the named multipart part from a
// captured request, so tests can assert what actually went on the wire.
func decodeDocumentPart(t *testing.T, req *http.Request, name string) textproto.MIMEHeader {
	t.Helper()
	_, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse media type: %v", err)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			t.Fatalf("part %q not found", name)
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		if part.FormName() == name {
			return part.Header
		}
	}
}

func newMimeTypeClient(t *testing.T) (*ym.Client, *testutil.FakeDoer) {
	t.Helper()
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`),
	}}

	return ym.NewClientWithHTTP(ym.Config{
		BaseURL:       "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1}},
	}, doer), doer
}

// MimeType must override the Content-Type of the document part. This capability
// came from the deleted files service and must survive consolidation.
func TestSendFileSetsMimeTypeOnDocumentPart(t *testing.T) {
	client, doer := newMimeTypeClient(t)
	_, err := NewService(client).SendFile(context.Background(), &SendFileRequest{
		ChatID:   ptrChat("c1"),
		Document: bytes.NewBufferString("%PDF-1.4"),
		Filename: "report.pdf",
		MimeType: "application/pdf",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := decodeDocumentPart(t, doer.Requests[0], "document").Get("Content-Type")
	if got != "application/pdf" {
		t.Fatalf("expected document part Content-Type application/pdf, got %q", got)
	}
}

// Without an explicit MimeType the document part must carry no Content-Type at
// all, leaving detection to the server rather than guessing a wrong type.
func TestSendFileOmitsContentTypeWhenMimeTypeEmpty(t *testing.T) {
	client, doer := newMimeTypeClient(t)
	_, err := NewService(client).SendFile(context.Background(), &SendFileRequest{
		ChatID:   ptrChat("c1"),
		Document: bytes.NewBufferString("data"),
		Filename: "f.txt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Checked by key presence, not Get: an empty Set value is indistinguishable
	// from an absent header through Get, which would hide a missing guard.
	if got, present := decodeDocumentPart(t, doer.Requests[0], "document")["Content-Type"]; present {
		t.Fatalf("expected no Content-Type header on document part, got %q", got)
	}
}

// The Bot API answers sendFile with flat fields: {"ok":true,"message_id":N,"file_id":"..."}.
// The deleted files service expected a nested "message" object instead and so could
// never succeed against the live API. This locks the documented shape in place.
func TestSendFileAcceptsDocumentedFlatResponse(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1647523230504005,"file_id":"abc"}`),
	}}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL:       "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1}},
	}, doer)

	msg, err := NewService(client).SendFile(context.Background(), &SendFileRequest{
		ChatID:   ptrChat("c1"),
		Document: bytes.NewBufferString("data"),
		Filename: "f.txt",
	})
	if err != nil {
		t.Fatalf("documented response shape must be accepted, got error: %v", err)
	}
	if msg == nil || msg.ID != 1647523230504005 {
		t.Fatalf("expected message_id 1647523230504005, got %v", msg)
	}
}

// decodeMultipartFields returns every non-file form field of a captured request.
func decodeMultipartFields(t *testing.T, req *http.Request) map[string]string {
	t.Helper()
	_, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse media type: %v", err)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	fields := map[string]string{}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return fields
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		if part.FileName() != "" {
			continue
		}
		val, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read part %q: %v", part.FormName(), err)
		}
		fields[part.FormName()] = string(val)
	}
}

// Every parameter the Bot API documents for sendFile must reach the wire.
func TestSendFileSendsAllDocumentedFields(t *testing.T) {
	client, doer := newMimeTypeClient(t)
	_, err := NewService(client).SendFile(context.Background(), &SendFileRequest{
		ChatID:              ptrChat("c1"),
		Document:            bytes.NewBufferString("data"),
		Filename:            "f.txt",
		ThreadID:            ym.Ptr(ym.ThreadID(11)),
		MessageID:           ym.Ptr(ym.MessageID(22)),
		ReplyMessageID:      ym.Ptr(ym.MessageID(33)),
		ReplyQuote:          "quoted",
		DisableNotification: ym.Ptr(true),
		Important:           ym.Ptr(true),
		ActionButtons: &ym.ActionButtons{Buttons: []ym.ActionButton{{
			Title: "Like",
			Icon:  ym.ActionButtonIcon{Type: ym.ActionButtonIconType, Value: ym.IconLike},
		}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fields := decodeMultipartFields(t, doer.Requests[0])
	for name, want := range map[string]string{
		"chat_id":              "c1",
		"thread_id":            "11",
		"message_id":           "22",
		"reply_message_id":     "33",
		"reply_quote":          "quoted",
		"disable_notification": "true",
		"important":            "true",
	} {
		if fields[name] != want {
			t.Errorf("field %q: want %q, got %q", name, want, fields[name])
		}
	}
	if !strings.Contains(fields["action_buttons"], `"title":"Like"`) {
		t.Errorf("action_buttons not sent as JSON: %q", fields["action_buttons"])
	}
}

// forwards is a documented parameter and must be serialized as JSON.
func TestSendFileSendsForwards(t *testing.T) {
	client, doer := newMimeTypeClient(t)
	_, err := NewService(client).SendFile(context.Background(), &SendFileRequest{
		ChatID:   ptrChat("c1"),
		Document: bytes.NewBufferString("data"),
		Filename: "f.txt",
		Forwards: []ym.Forward{{ChatID: "src", MessageIDs: []ym.MessageID{1, 2}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := decodeMultipartFields(t, doer.Requests[0])["forwards"]
	if !strings.Contains(got, `"chat_id":"src"`) || !strings.Contains(got, `[1,2]`) {
		t.Errorf("unexpected forwards payload: %q", got)
	}
}

// sendFile answers with a file_id that the caller must be able to read back.
func TestSendFileReturnsFileID(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":7,"file_id":"fid-1"}`),
	}}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL:       "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1}},
	}, doer)

	msg, err := NewService(client).SendFile(context.Background(), &SendFileRequest{
		ChatID: ptrChat("c1"), Document: bytes.NewBufferString("d"), Filename: "f.txt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Document == nil || msg.Document.ID != "fid-1" {
		t.Fatalf("expected Document.ID fid-1, got %+v", msg.Document)
	}
}

func newFakeClient(t *testing.T, body string) (*ym.Client, *testutil.FakeDoer) {
	t.Helper()
	doer := &testutil.FakeDoer{Responses: []*http.Response{testutil.NewResponse(http.StatusOK, body)}}

	return ym.NewClientWithHTTP(ym.Config{
		BaseURL:       "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1}},
	}, doer), doer
}

// sendImage answers with file_id plus the stored dimensions.
func TestSendImageReturnsFileIDAndDimensions(t *testing.T) {
	client, _ := newFakeClient(t, `{"ok":true,"message_id":7,"file_id":"fid-i","width":1920,"height":1080}`)
	msg, err := NewService(client).SendImage(context.Background(), &SendImageRequest{
		ChatID: ptrChat("c1"), Image: bytes.NewBufferString("png"), Filename: "p.png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Image == nil || msg.Image.FileID != "fid-i" || msg.Image.Width != 1920 || msg.Image.Height != 1080 {
		t.Fatalf("expected image fid-i 1920x1080, got %+v", msg.Image)
	}
}

// sendGallery answers with one entry per uploaded image.
func TestSendGalleryReturnsImages(t *testing.T) {
	client, _ := newFakeClient(t,
		`{"ok":true,"message_id":7,"images":[{"file_id":"a","width":1920,"height":1080},{"file_id":"b","width":1280,"height":720}]}`)
	msg, err := NewService(client).SendGallery(context.Background(), &SendGalleryRequest{
		ChatID: ptrChat("c1"),
		Images: []FilePart{{Reader: bytes.NewBufferString("1"), Filename: "a.png"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Gallery) != 2 || msg.Gallery[0].FileID != "a" || msg.Gallery[1].Height != 720 {
		t.Fatalf("unexpected gallery: %+v", msg.Gallery)
	}
}

// The shared parameters must reach the wire for every multipart method, not
// just sendFile — that asymmetry is what this package used to suffer from.
func TestSendImageAndGallerySendCommonFields(t *testing.T) {
	common := func(fields map[string]string) {
		t.Helper()
		for name, want := range map[string]string{"reply_message_id": "33", "important": "true"} {
			if fields[name] != want {
				t.Errorf("field %q: want %q, got %q", name, want, fields[name])
			}
		}
	}

	imgClient, imgDoer := newFakeClient(t, `{"ok":true,"message_id":1}`)
	if _, err := NewService(imgClient).SendImage(context.Background(), &SendImageRequest{
		ChatID: ptrChat("c1"), Image: bytes.NewBufferString("p"), Filename: "p.png",
		ReplyMessageID: ym.Ptr(ym.MessageID(33)), Important: ym.Ptr(true),
	}); err != nil {
		t.Fatalf("sendImage: %v", err)
	}
	common(decodeMultipartFields(t, imgDoer.Requests[0]))

	galClient, galDoer := newFakeClient(t, `{"ok":true,"message_id":1}`)
	if _, err := NewService(galClient).SendGallery(context.Background(), &SendGalleryRequest{
		ChatID: ptrChat("c1"), Images: []FilePart{{Reader: bytes.NewBufferString("1"), Filename: "a.png"}},
		ReplyMessageID: ym.Ptr(ym.MessageID(33)), Important: ym.Ptr(true),
	}); err != nil {
		t.Fatalf("sendGallery: %v", err)
	}
	common(decodeMultipartFields(t, galDoer.Requests[0]))
}

// Constraints the Bot API documents are rejected before a request is spent.
func TestDocumentedConstraintsRejectedLocally(t *testing.T) {
	tests := []struct {
		name string
		req  *SendFileRequest
		want error
	}{
		{
			name: "reply_quote without reply_message_id",
			req:  &SendFileRequest{ChatID: ptrChat("c1"), ReplyQuote: "q"},
			want: ErrReplyQuoteNeedsReply,
		},
		{
			name: "forwards combined with reply",
			req: &SendFileRequest{
				ChatID: ptrChat("c1"), ReplyMessageID: ym.Ptr(ym.MessageID(1)),
				Forwards: []ym.Forward{{ChatID: "src", MessageIDs: []ym.MessageID{1}}},
			},
			want: ErrForwardsWithReply,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.req.Document = bytes.NewBufferString("d")
			tt.req.Filename = "f.txt"
			_, err := NewService(nil).SendFile(context.Background(), tt.req)
			if !errors.Is(err, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, err)
			}
		})
	}

	// Limits are reported as a typed error so callers can match every
	// documented violation the same way.
	t.Run("more than six action buttons", func(t *testing.T) {
		_, err := NewService(nil).SendFile(context.Background(), &SendFileRequest{
			ChatID:        ptrChat("c1"),
			Document:      bytes.NewBufferString("d"),
			Filename:      "f.txt",
			ActionButtons: &ym.ActionButtons{Buttons: make([]ym.ActionButton, 7)},
		})

		var limitErr *ym.LimitError
		if !errors.As(err, &limitErr) {
			t.Fatalf("expected a *ym.LimitError, got %T (%v)", err, err)
		}
		if limitErr.Value != 7 || limitErr.Limit != ym.MaxActionButtons {
			t.Fatalf("unexpected limit error: %#v", limitErr)
		}
	})
}

// The flat layout must survive the real send paths, not just a direct
// json.Marshal of the type: multipart writes the field itself.
func TestFlatSuggestButtonsReachTheWire(t *testing.T) {
	buttons := &ym.SuggestButtons{
		Layout:  ym.Ptr(ym.SuggestLayoutFlat),
		Buttons: [][]ym.InlineSuggestButton{{{Title: "a"}}, {{Title: "b"}}},
	}

	fileClient, fileDoer := newFakeClient(t, `{"ok":true,"message_id":1}`)
	if _, err := NewService(fileClient).SendFile(context.Background(), &SendFileRequest{
		ChatID: ptrChat("c1"), Document: bytes.NewBufferString("d"), Filename: "f.txt",
		SuggestButtons: buttons,
	}); err != nil {
		t.Fatalf("sendFile: %v", err)
	}
	got := decodeMultipartFields(t, fileDoer.Requests[0])["suggest_buttons"]
	if !strings.Contains(got, `"buttons":[{"title":"a"},{"title":"b"}]`) {
		t.Errorf("multipart suggest_buttons not flat: %s", got)
	}

	textClient, textDoer := newFakeClient(t, `{"ok":true,"message_id":1}`)
	if _, err := NewService(textClient).SendToChat(context.Background(), "c1", "hi", &SendMessageOptions{
		SuggestButtons: buttons,
	}); err != nil {
		t.Fatalf("sendText: %v", err)
	}
	body, err := io.ReadAll(textDoer.Requests[0].Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), `"buttons":[{"title":"a"},{"title":"b"}]`) {
		t.Errorf("sendText suggest_buttons not flat: %s", body)
	}
}
