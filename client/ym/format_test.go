package ym

import (
	"strings"
	"testing"
)

func TestEscapeMarkdown(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain text is untouched", "hello world", "hello world"},
		{"bold markers", "**bold**", `\*\*bold\*\*`},
		{"italic markers", "__x__", `\_\_x\_\_`},
		{"link syntax", "[a](b)", `\[a\]\(b\)`},
		{"backslash itself", `a\b`, `a\\b`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EscapeMarkdown(tc.in); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("keeps multi-byte text intact", func(t *testing.T) {
		if got := EscapeMarkdown("привет"); got != "привет" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestFormattingHelpers(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"bold", Bold("x"), "**x**"},
		{"italic", Italic("x"), "__x__"},
		{"strikethrough", Strikethrough("x"), "~~x~~"},
		{"underline", Underline("x"), "++x++"},
		{"code", Code("x"), "`x`"},
		{"link", Link("label", "https://example.com"), "[label](https://example.com)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

// Inline code is the reliable way to render arbitrary input, so a stray
// backtick must not be able to end the span early.
func TestCodeDropsBackticks(t *testing.T) {
	if got := Code("a`b"); strings.Count(got, "`") != 2 {
		t.Fatalf("a backtick escaped the span: %q", got)
	}
}

func TestCodeBlock(t *testing.T) {
	t.Run("carries the language", func(t *testing.T) {
		if got := CodeBlock("go", "x := 1"); got != "```go\nx := 1\n```" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("drops a nested fence", func(t *testing.T) {
		if got := CodeBlock("", "a```b"); strings.Count(got, "```") != 2 {
			t.Fatalf("a fence escaped the block: %q", got)
		}
	})
}

func TestLinkEscapesDelimiters(t *testing.T) {
	got := Link("a]b", "http://x/(y)")
	if got != `[a\]b](http://x/\(y\))` {
		t.Fatalf("got %q", got)
	}
}
