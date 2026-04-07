package messages

import (
	"bytes"
	"context"
	"errors"
	"net/http"
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
	}
	for _, tc := range tests {
		got := sanitizeFilename(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func ptrChat(id ym.ChatID) *ym.ChatID { return &id }
