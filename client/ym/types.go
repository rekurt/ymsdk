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
	// SetID identifies the sticker set. Together with ID it is what
	// sendSticker needs to resend the sticker.
	SetID string `json:"set_id,omitempty"`
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
//
// Not every update describes a message: reaction and membership events arrive
// with Reaction or ChatMembersUpdate set and no text, so always check the field
// you intend to read before using it.
type Update struct {
	UpdateID  int64     `json:"update_id"`
	Chat      *Chat     `json:"chat,omitempty"`
	From      *Sender   `json:"from,omitempty"`
	Text      string    `json:"text,omitempty"`
	Timestamp int64     `json:"timestamp,omitempty"`
	MessageID MessageID `json:"message_id,omitempty"`
	ThreadID  *ThreadID `json:"thread_id,omitempty"`
	Sticker   *Sticker  `json:"sticker,omitempty"`

	// Document is the attached file. The API names this field "file".
	Document *File `json:"file,omitempty"`

	// Images holds the attached images. The outer slice has one entry per
	// image; the inner slice holds that image's size variants, from the
	// smallest preview to the original. Use [Update.OriginalImages] to get one
	// entry per image at full size.
	Images [][]Image `json:"images,omitempty"`

	// ForwardedMessages are the messages forwarded into this chat.
	ForwardedMessages []Update `json:"forwarded_messages,omitempty"`

	// ReplyToMessage is the message this one replies to, if any.
	ReplyToMessage *Update `json:"reply_to_message,omitempty"`

	// ChatMembersUpdate describes a change to a group chat's membership.
	// Delivered only while the bot's get_members_changed setting is on, and
	// never for channels.
	ChatMembersUpdate *ChatMembersUpdate `json:"chat_members_update,omitempty"`

	// Reaction describes a reaction being added or removed. Delivered only
	// while the bot's get_reactions setting is on.
	Reaction *ReactionEvent `json:"reaction,omitempty"`

	// BotRequest carries callback data from an interactive button press.
	BotRequest *BotRequest `json:"bot_request,omitempty"`

	// Image is not part of the documented update schema; the API sends image
	// attachments in Images.
	//
	// Deprecated: read [Update.Images] or [Update.OriginalImages] instead.
	Image *Image `json:"-"`

	// Forward is a response-shaped forward descriptor that the update schema
	// does not use.
	//
	// Deprecated: read [Update.ForwardedMessages] instead.
	Forward *ForwardInfo `json:"-"`
}

// OriginalImages returns one entry per attached image, picking the original
// full-size variant of each.
//
// The API delivers every image as a list of size variants — small, middle,
// middle-400 and the original — and marks the original by carrying its file
// name and byte size. When no variant is marked, the widest one is used.
func (u *Update) OriginalImages() []Image {
	if u == nil || len(u.Images) == 0 {
		return nil
	}

	out := make([]Image, 0, len(u.Images))
	for _, variants := range u.Images {
		if len(variants) == 0 {
			continue
		}
		best := variants[0]
		for _, v := range variants[1:] {
			if v.Name != "" || v.Size > 0 {
				best = v

				break
			}
			if v.Width > best.Width {
				best = v
			}
		}
		out = append(out, best)
	}

	return out
}

// ChatMembersUpdate describes users joining or leaving a group chat.
type ChatMembersUpdate struct {
	NewChatMembers     []Sender `json:"new_chat_members,omitempty"`
	RemovedChatMembers []Sender `json:"removed_chat_members,omitempty"`
}

// ReactionEvent describes a reaction a user added to or removed from a message.
type ReactionEvent struct {
	MessageID MessageID      `json:"message_id"`
	Reaction  Reaction       `json:"reaction"`
	Action    ReactionAction `json:"action"`
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
		Gallery:   u.OriginalImages(),
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

// BotSelf contains information about the bot itself, returned by the self
// endpoints.
type BotSelf struct {
	ID            string    `json:"id"`
	DisplayName   string    `json:"display_name"`
	WebhookURL    *string   `json:"webhook_url,omitempty"`
	Organizations []int64   `json:"organizations,omitempty"`
	Login         UserLogin `json:"login"`
	// Settings reports which optional update types the bot receives.
	Settings *BotSettings `json:"settings,omitempty"`
}

// ParseTime converts RFC3339 time strings to time.Time if needed by consumers.
func (m *Message) ParseTime() (time.Time, error) {
	return time.Parse(time.RFC3339, m.CreatedAt)
}

// UserID is a user's UUID. Several endpoints accept it as an alternative to a
// login when addressing a private chat.
type UserID string

// Target identifies the recipient of an operation. Exactly one field must be
// set — the API rejects requests carrying more than one.
//
// Target is embedded into request bodies, so its fields marshal inline.
// Build one with [ChatTarget], [LoginTarget] or [UserIDTarget].
type Target struct {
	ChatID ChatID    `json:"chat_id,omitempty"`
	Login  UserLogin `json:"login,omitempty"`
	UserID UserID    `json:"user_id,omitempty"`
}

// ChatTarget addresses a group chat or channel by its identifier.
func ChatTarget(id ChatID) Target { return Target{ChatID: id} }

// LoginTarget addresses a user's private chat by login.
func LoginTarget(login UserLogin) Target { return Target{Login: login} }

// UserIDTarget addresses a user's private chat by UUID.
func UserIDTarget(id UserID) Target { return Target{UserID: id} }

// Reaction identifies an emoji reaction. The reactions a chat allows are listed
// in ChatInfo.AvailableReactions.
type Reaction struct {
	// Type is "default_reaction" for the standard set.
	Type string `json:"type"`
	// Name is the reaction's name, for example "like" or "fire".
	Name string `json:"name"`
}

// ReactionTypeDefault is the Type value used by the standard reaction set.
const ReactionTypeDefault = "default_reaction"

// DefaultReaction builds a Reaction from the standard set.
func DefaultReaction(name string) *Reaction {
	return &Reaction{Type: ReactionTypeDefault, Name: name}
}

// ReactionCount is an anonymous per-reaction tally, returned for channels.
type ReactionCount struct {
	Reaction Reaction `json:"reaction"`
	Count    int      `json:"count"`
}

// MessageReactionEntry is a single reaction together with its author, returned
// for private and group chats.
type MessageReactionEntry struct {
	Reaction  Reaction `json:"reaction"`
	Timestamp int64    `json:"timestamp"`
	User      Sender   `json:"user"`
}

// ReactionAction distinguishes a reaction being added from one being removed.
type ReactionAction string

const (
	// ReactionAdded means a user set the reaction.
	ReactionAdded ReactionAction = "add"
	// ReactionRemoved means a user cleared the reaction.
	ReactionRemoved ReactionAction = "remove"
)

// Forward references messages to be forwarded into the target chat.
// It cannot be combined with a reply in the same request.
type Forward struct {
	ChatID     ChatID      `json:"chat_id"`
	MessageIDs []MessageID `json:"message_ids"`
}

// ActionButton is a single action button rendered under a message.
type ActionButton struct {
	ID         string      `json:"id,omitempty"`
	Title      string      `json:"title,omitempty"`
	Directives []Directive `json:"directives,omitempty"`
}

// ActionButtons is a row of action buttons under a message. At most 6.
type ActionButtons struct {
	Buttons []ActionButton `json:"buttons"`
}

// Button is an inline keyboard button.
//
// Deprecated: the API marks Button and inline_keyboard as obsolete. Use
// [SuggestButtons] instead.
type Button struct {
	Text         string          `json:"text"`
	CallbackData json.RawMessage `json:"callback_data,omitempty"`
	URL          string          `json:"url,omitempty"`
}

// SharedFile references a file already uploaded to the messenger, so it can be
// sent again without re-uploading its bytes.
type SharedFile struct {
	FileID string `json:"file_id"`
}

// SharedImage references an image already uploaded to the messenger. Width and
// height are required by the API and come from the original upload or update.
type SharedImage struct {
	FileID string `json:"file_id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// TypingType selects which indicator sendTyping displays.
type TypingType string

const (
	// TypingText shows the ordinary "typing…" indicator. This is the default.
	TypingText TypingType = "text"
	// TypingProcessing shows a processing indicator. Private chats only.
	TypingProcessing TypingType = "processing"
)

// ProcessingDisplay selects how a processing indicator is rendered.
type ProcessingDisplay string

const (
	// ProcessingDisplayDefault shows the standard processing indicator.
	ProcessingDisplayDefault ProcessingDisplay = "default"
	// ProcessingDisplayText shows caller-supplied text.
	ProcessingDisplayText ProcessingDisplay = "text"
)

// ProcessingContent configures a processing indicator. Required when the
// indicator type is [TypingProcessing].
type ProcessingContent struct {
	Display ProcessingDisplay `json:"display"`
	// Text is required when Display is [ProcessingDisplayText]; 1 to 100 characters.
	Text string `json:"text,omitempty"`
}

// ReactionsType tells which shape a getReactions response took.
type ReactionsType string

const (
	// ReactionsPublic is returned for private and group chats: reactions carry
	// their authors and are paginated.
	ReactionsPublic ReactionsType = "public"
	// ReactionsPrivate is returned for channels: anonymous per-reaction tallies.
	ReactionsPrivate ReactionsType = "private"
)

// ReactionsPage is the polymorphic result of getReactions. Read [ReactionsPage.Type]
// first: List is populated for [ReactionsPublic] and Counts for [ReactionsPrivate].
type ReactionsPage struct {
	Type   ReactionsType          `json:"reactions_type"`
	List   []MessageReactionEntry `json:"reactions_list,omitempty"`
	Counts []ReactionCount        `json:"reactions_count,omitempty"`
}

// ChatMemberRole is a participant's role in a chat or channel.
type ChatMemberRole string

const (
	// RoleAdmin is a chat or channel administrator.
	RoleAdmin ChatMemberRole = "admin"
	// RoleMember is an ordinary chat participant.
	RoleMember ChatMemberRole = "member"
	// RoleSubscriber is a channel subscriber.
	RoleSubscriber ChatMemberRole = "subscriber"
)

// ChatMember describes a participant of a chat or channel.
type ChatMember struct {
	// GUID uniquely identifies the participant and is the pagination cursor
	// for the member list.
	GUID  string         `json:"guid"`
	Login UserLogin      `json:"login,omitempty"`
	Role  ChatMemberRole `json:"role"`
	IsBot bool           `json:"is_bot"`
}

// ChatMetaData describes a chat or channel in a chat listing.
type ChatMetaData struct {
	Type ChatType `json:"type"`
	// ID is also the pagination cursor for the chat list.
	ID          ChatID `json:"id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	// Username is the other participant's login, set only for private chats.
	Username string `json:"username,omitempty"`
}

// ChatInfo describes a single chat or channel in detail.
type ChatInfo struct {
	Type        ChatType `json:"type"`
	ID          ChatID   `json:"id"`
	Private     bool     `json:"private,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	InviteHash  string   `json:"invite_hash,omitempty"`
	InviteLink  string   `json:"invite_link,omitempty"`
	// AvailableReactions lists the reactions this chat permits. Pass one of
	// these to the reaction methods.
	AvailableReactions []Reaction `json:"available_reactions,omitempty"`
}

// BotSettings are the bot's own feature flags. Both are off by default, and
// the corresponding update types are not delivered until they are switched on.
type BotSettings struct {
	// GetReactions enables reaction events on updates.
	GetReactions *bool `json:"get_reactions,omitempty"`
	// GetMembersChanged enables group membership events on updates.
	GetMembersChanged *bool `json:"get_members_changed,omitempty"`
}
