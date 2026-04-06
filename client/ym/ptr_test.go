package ym

import (
	"testing"
)

func TestPtr(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		p := Ptr(42)
		if *p != 42 {
			t.Fatalf("expected 42, got %d", *p)
		}
	})

	t.Run("string", func(t *testing.T) {
		p := Ptr("hello")
		if *p != "hello" {
			t.Fatalf("expected hello, got %s", *p)
		}
	})

	t.Run("bool", func(t *testing.T) {
		p := Ptr(true)
		if !*p {
			t.Fatal("expected true")
		}
	})

	t.Run("ChatID", func(t *testing.T) {
		p := Ptr(ChatID("chat-1"))
		if *p != "chat-1" {
			t.Fatalf("expected chat-1, got %s", *p)
		}
	})

	t.Run("zero value", func(t *testing.T) {
		p := Ptr(0)
		if *p != 0 {
			t.Fatalf("expected 0, got %d", *p)
		}
	})
}
