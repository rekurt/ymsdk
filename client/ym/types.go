package ym

import (
	"encoding/json"
	"time"
)

// ChatType represents the type of a Yandex Messenger chat.
type ChatType string

const (
	// ChatTypePrivate is a one-on-one private conversation.
	ChatTypePrivate ChatType = "private"
	// ChatTypeGroup is a group chat with multiple participants.
	ChatTypeGroup ChatType = "group"
	// ChatTypeChannel is a broadcast channel with subscribers.
	ChatTypeChannel ChatType = "channel"
)

// ChatID is a unique identifier for a chat.
type ChatID string

// UserLogin is a Yandex Messenger user login (e.g. "john.doe").
type UserLogin string

// MessageID is a unique identifier for a message within a chat.
type MessageID int64

// ThreadID is a unique identifier for a message thread.
type ThreadID int64

// Chat represents a Yandex Messenger chat (private, group, or channel).
type Chat struct {
	ID             ChatID   `json:"id"`
	Type           ChatType `json:"type"`
	OrganizationID string   `json:"organization_id,omitempty"`
	Title          string   `json:"title,omitempty"`
	Description    string   `json:"description,omitempty"`
	IsChannel      bool     `json:"is_channel,omitempty"`
}

// Sender contains information about the user who sent a message.
type Sender struct {
	ID          string    `json:"id,omitempty"`
	Login       UserLogin `json:"login"`
	Name        string    `json:"name,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Robot       *bool     `json:"robot,omitempty"`
}

// ForwardInfo contains metadata about a forwarded message.
type ForwardInfo struct {
	From      *Sender   `json:"from,omitempty"`
	Chat      *Chat     `json:"chat,omitempty"`
	MessageID MessageID `json:"message_id,omitempty"`
}

// Sticker represents a sticker attachment in a message.
type Sticker struct {
	ID    string `json:"id,omitempty"`
	Emoji string `json:"emoji,omitempty"`
}

// Image represents an image attachment in a message.
type Image struct {
	FileID string `json:"file_id,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Name   string `json:"name,omitempty"`
}

// File represents a document attachment in a message.
type File struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
	URL      string `json:"url,omitempty"`
}

// Message represents a message returned by the Yandex Messenger API.
type Message struct {
	ID        MessageID    `json:"message_id"`
	Chat      Chat         `json:"chat"`
	From      Sender       `json:"from"`
	Text      string       `json:"text,omitempty"`
	CreatedAt string       `json:"created_at,omitempty"`
	Timestamp int64        `json:"timestamp,omitempty"`
	ThreadID  *ThreadID    `json:"thread_id,omitempty"`
	Forward   *ForwardInfo `json:"forward,omitempty"`
	Sticker   *Sticker     `json:"sticker,omitempty"`
	Image     *Image       `json:"image,omitempty"`
	Gallery   []Image      `json:"gallery,omitempty"`
	Document  *File        `json:"document,omitempty"`
}

// Update represents an incoming update from the getUpdates endpoint.
type Update struct {
	UpdateID   int64        `json:"update_id"`
	Chat       *Chat        `json:"chat,omitempty"`
	From       *Sender      `json:"from,omitempty"`
	Text       string       `json:"text,omitempty"`
	Timestamp  int64        `json:"timestamp,omitempty"`
	MessageID  MessageID    `json:"message_id,omitempty"`
	ThreadID   *ThreadID    `json:"thread_id,omitempty"`
	Forward    *ForwardInfo `json:"forward,omitempty"`
	Sticker    *Sticker     `json:"sticker,omitempty"`
	Image      *Image       `json:"image,omitempty"`
	Images     []Image      `json:"images,omitempty"`
	Document   *File        `json:"document,omitempty"`
	BotRequest *BotRequest  `json:"bot_request,omitempty"`
}

// ToMessage converts an Update to a Message by promoting its fields.
// This is useful for code that expects a Message struct.
func (u *Update) ToMessage() *Message {
	if u == nil || u.Chat == nil || u.From == nil {
		return nil
	}

	return &Message{
		ID:        u.MessageID,
		Chat:      *u.Chat,
		From:      *u.From,
		Text:      u.Text,
		Timestamp: u.Timestamp,
		ThreadID:  u.ThreadID,
		Forward:   u.Forward,
		Sticker:   u.Sticker,
		Image:     u.Image,
		Gallery:   u.Images,
		Document:  u.Document,
	}
}

// DirectiveType constants for button actions.
const (
	DirectiveOpenURI          = "open_uri"
	DirectiveSendMessage      = "send_message"
	DirectiveServerAction     = "server_action"
	DirectiveSetElementsState = "set_elements_state"
)

// Directive describes an action triggered by a button press.
type Directive struct {
	Type           string          `json:"type"`
	URI            string          `json:"uri,omitempty"`
	Text           string          `json:"text,omitempty"`
	Name           string          `json:"name,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	IDs            []string        `json:"ids,omitempty"`
	State          string          `json:"state,omitempty"`
	TimeoutSeconds *int            `json:"timeout_seconds,omitempty"`
}

// InlineSuggestButton is a single button in a SuggestButtons keyboard.
type InlineSuggestButton struct {
	ID         string      `json:"id,omitempty"`
	Title      string      `json:"title,omitempty"`
	Directives []Directive `json:"directives,omitempty"`
}

// SuggestButtons is a keyboard of interactive buttons attached to a message.
type SuggestButtons struct {
	Layout  *string                 `json:"layout,omitempty"`
	Persist *bool                   `json:"persist,omitempty"`
	Buttons [][]InlineSuggestButton `json:"buttons"`
}

// ServerAction represents a callback action triggered by a button.
type ServerAction struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// BotRequestError describes an error that occurred processing a button directive.
type BotRequestError struct {
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
}

// BotRequest contains callback data from interactive button presses.
type BotRequest struct {
	ServerAction *ServerAction     `json:"server_action,omitempty"`
	ElementID    string            `json:"element_id,omitempty"`
	Errors       []BotRequestError `json:"errors,omitempty"`
}

// UserRef identifies a user by login, used in member lists.
type UserRef struct {
	Login UserLogin `json:"login"`
}

// UserLink contains deep links for starting a chat or call with a user.
type UserLink struct {
	ID       string `json:"id"`
	ChatLink string `json:"chat_link"`
	CallLink string `json:"call_link"`
}

// PollResult holds aggregated poll voting results.
type PollResult struct {
	VotedCount int         `json:"voted_count"`
	Answers    map[int]int `json:"answers"`
}

// Vote represents a single vote cast in a poll.
type Vote struct {
	Timestamp int64   `json:"timestamp"`
	User      UserRef `json:"user"`
}

// PollVotersPage is a paginated response of poll voters for a specific answer.
type PollVotersPage struct {
	AnswerID   int    `json:"answer_id"`
	VotedCount int    `json:"voted_count"`
	Cursor     int64  `json:"cursor"`
	Votes      []Vote `json:"votes"`
}

// BotSelf contains information about the bot itself, returned by the self.update endpoint.
type BotSelf struct {
	ID            string    `json:"id"`
	DisplayName   string    `json:"display_name"`
	WebhookURL    *string   `json:"webhook_url,omitempty"`
	Organizations []int64   `json:"organizations,omitempty"`
	Login         UserLogin `json:"login"`
}

// ParseTime converts RFC3339 time strings to time.Time if needed by consumers.
func (m *Message) ParseTime() (time.Time, error) {
	return time.Parse(time.RFC3339, m.CreatedAt)
}
