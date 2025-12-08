package ym

import "time"

type ChatType string

const (
	ChatTypePrivate ChatType = "private"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type ChatID string

type UserLogin string

type MessageID int64

type ThreadID int64

type Chat struct {
	ID             ChatID   `json:"id"`
	Type           ChatType `json:"type"`
	OrganizationID string   `json:"organization_id,omitempty"`
	Title          string   `json:"title,omitempty"`
	Description    string   `json:"description,omitempty"`
	IsChannel      bool     `json:"is_channel,omitempty"`
}

type Sender struct {
	Login UserLogin `json:"login"`
	Name  string    `json:"name,omitempty"`
}

type ForwardInfo struct {
	From      *Sender   `json:"from,omitempty"`
	Chat      *Chat     `json:"chat,omitempty"`
	MessageID MessageID `json:"message_id,omitempty"`
}

type Sticker struct {
	ID    string `json:"id,omitempty"`
	Emoji string `json:"emoji,omitempty"`
}

type Image struct {
	ID     string `json:"id,omitempty"`
	URL    string `json:"url,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type File struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
	URL      string `json:"url,omitempty"`
}

type Message struct {
	ID        MessageID    `json:"id"`
	Chat      Chat         `json:"chat"`
	From      Sender       `json:"from"`
	Text      string       `json:"text,omitempty"`
	CreatedAt string       `json:"created_at"`
	ThreadID  *ThreadID    `json:"thread_id,omitempty"`
	Forward   *ForwardInfo `json:"forward,omitempty"`
	Sticker   *Sticker     `json:"sticker,omitempty"`
	Image     *Image       `json:"image,omitempty"`
	Gallery   []Image      `json:"gallery,omitempty"`
	Document  *File        `json:"document,omitempty"`
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

type UserRef struct {
	Login UserLogin `json:"login"`
}

type UserLink struct {
	ID       string `json:"id"`
	ChatLink string `json:"chat_link"`
	CallLink string `json:"call_link"`
}

type PollResult struct {
	VotedCount int         `json:"voted_count"`
	Answers    map[int]int `json:"answers"`
}

type Vote struct {
	Timestamp int64   `json:"timestamp"`
	User      UserRef `json:"user"`
}

type PollVotersPage struct {
	AnswerID   int    `json:"answer_id"`
	VotedCount int    `json:"voted_count"`
	Cursor     int64  `json:"cursor"`
	Votes      []Vote `json:"votes"`
}

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
