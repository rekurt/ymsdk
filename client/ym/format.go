package ym

import "strings"

// Text formatting helpers for the messenger's markdown-like syntax:
// **bold**, __italic__, ~~strikethrough~~, ++underline++, `code`,
// ```language fenced blocks``` and [text](url).

// markdownMarkers are the characters that start a formatting sequence.
const markdownMarkers = "*_~+`[]()\\"

// EscapeMarkdown neutralises formatting markers in text that came from a user,
// so an incoming message containing ** or __ cannot reshape the reply's layout.
//
// The API documents the markup but not an escape syntax, so this applies the
// usual convention of prefixing each marker with a backslash. Where absolute
// certainty matters — echoing arbitrary input, rendering identifiers — prefer
// [Code], which suppresses formatting outright.
func EscapeMarkdown(text string) string {
	var b strings.Builder
	b.Grow(len(text) + len(text)/8)

	for _, r := range text {
		if strings.ContainsRune(markdownMarkers, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}

	return b.String()
}

// Bold wraps text in the bold markers.
func Bold(text string) string { return "**" + text + "**" }

// Italic wraps text in the italic markers.
func Italic(text string) string { return "__" + text + "__" }

// Strikethrough wraps text in the strikethrough markers.
func Strikethrough(text string) string { return "~~" + text + "~~" }

// Underline wraps text in the underline markers.
func Underline(text string) string { return "++" + text + "++" }

// Code renders text as inline code, which suppresses any formatting inside it.
// Backticks in text are dropped, since the syntax offers no way to escape them.
func Code(text string) string {
	return "`" + strings.ReplaceAll(text, "`", "") + "`"
}

// CodeBlock renders text as a fenced block. Pass an empty language for a plain
// block. A fence appearing inside text is dropped, as it would end the block.
func CodeBlock(language, text string) string {
	cleaned := strings.ReplaceAll(text, "```", "")

	return "```" + language + "\n" + cleaned + "\n```"
}

// Link renders a labelled hyperlink, escaping the characters that would
// otherwise close the label or the URL early.
func Link(text, url string) string {
	label := strings.NewReplacer("[", `\[`, "]", `\]`).Replace(text)
	href := strings.NewReplacer("(", `\(`, ")", `\)`).Replace(url)

	return "[" + label + "](" + href + ")"
}
