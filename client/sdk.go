package client

import (
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/chats"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/polls"
	"github.com/rekurt/ymsdk/client/ym/self"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/users"
)

// YMClient is a convenience aggregator that holds pre-constructed service instances.
// Use [New] to create one from a [ym.Config], or [Wrap] to wrap an existing [ym.Client].
type YMClient struct {
	Client   *ym.Client
	Messages *messages.Service
	Chats    *chats.Service
	Users    *users.Service
	Polls    *polls.Service
	Updates  *updates.Service
	Self     *self.Service
}

// New creates a YMClient with a new default HTTP client.
func New(cfg ym.Config) *YMClient {
	cl := ym.NewClient(cfg)

	return Wrap(cl)
}

// Wrap wraps an existing ym.Client into a YMClient with all services initialized.
func Wrap(cl *ym.Client) *YMClient {
	return &YMClient{
		Client:   cl,
		Messages: messages.NewService(cl),
		Chats:    chats.NewService(cl),
		Users:    users.NewService(cl),
		Polls:    polls.NewService(cl),
		Updates:  updates.NewService(cl),
		Self:     self.NewService(cl),
	}
}
