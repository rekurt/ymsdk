package ym

import (
	"testing"
	"time"
)

func TestNextBackoff(t *testing.T) {
	t.Run("doubles backoff", func(t *testing.T) {
		got := NextBackoff(500*time.Millisecond, 10*time.Second)
		if got != time.Second {
			t.Fatalf("expected 1s, got %v", got)
		}
	})

	t.Run("caps at maximum", func(t *testing.T) {
		got := NextBackoff(8*time.Second, 10*time.Second)
		if got != 10*time.Second {
			t.Fatalf("expected 10s, got %v", got)
		}
	})

	t.Run("zero current uses default", func(t *testing.T) {
		got := NextBackoff(0, 10*time.Second)
		if got != time.Second {
			t.Fatalf("expected 1s, got %v", got)
		}
	})

	t.Run("no maximum", func(t *testing.T) {
		got := NextBackoff(time.Second, 0)
		if got != 2*time.Second {
			t.Fatalf("expected 2s, got %v", got)
		}
	})
}

func TestShouldRetryHTTP(t *testing.T) {
	list := []int{500, 502, 503, 504}

	t.Run("retryable status", func(t *testing.T) {
		if !ShouldRetryHTTP(502, list) {
			t.Fatal("expected 502 to be retryable")
		}
	})

	t.Run("non-retryable status", func(t *testing.T) {
		if ShouldRetryHTTP(400, list) {
			t.Fatal("expected 400 to not be retryable")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		if ShouldRetryHTTP(500, nil) {
			t.Fatal("expected false with nil list")
		}
	})
}
