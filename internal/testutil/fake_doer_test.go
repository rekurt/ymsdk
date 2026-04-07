package testutil

import (
	"net/http"
	"sync"
	"testing"
)

func TestFakeDoerConcurrentAccess(t *testing.T) {
	t.Parallel()

	responses := make([]*http.Response, 100)
	for i := range responses {
		responses[i] = &http.Response{StatusCode: http.StatusOK}
	}

	fd := &FakeDoer{Responses: responses}

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
			_, _ = fd.Do(req)
		}()
	}
	wg.Wait()

	if fd.CallCount() != 100 {
		t.Fatalf("expected 100 calls, got %d", fd.CallCount())
	}
}

func TestFakeDoerReset(t *testing.T) {
	fd := &FakeDoer{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK},
		},
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, _ = fd.Do(req)
	if fd.CallCount() != 1 {
		t.Fatalf("expected 1 call, got %d", fd.CallCount())
	}
	fd.Reset()
	if fd.CallCount() != 0 {
		t.Fatalf("expected 0 calls after reset, got %d", fd.CallCount())
	}
}
