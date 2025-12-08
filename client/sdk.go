package client

import (
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/chats"
	"github.com/rekurt/ymsdk/client/ym/files"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/polls"
	"github.com/rekurt/ymsdk/client/ym/self"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/users"
)

type YMClient struct {
	Client   *ym.Client
	Messages *messages.Service
	Files    *files.Service
	Chats    *chats.Service
	Users    *users.Service
	Polls    *polls.Service
	Updates  *updates.Service
	Self     *self.Service
}

// New создает YMClient с новым HTTP-клиентом.
func New(cfg ym.Config) *YMClient {
	cl := ym.NewClient(cfg)

	return Wrap(cl)
}

// Wrap оборачивает уже созданный ym.Client в YMClient.
func Wrap(cl *ym.Client) *YMClient {
	return &YMClient{
		Client:   cl,
		Messages: messages.NewService(cl),
		Files:    files.NewService(cl),
		Chats:    chats.NewService(cl),
		Users:    users.NewService(cl),
		Polls:    polls.NewService(cl),
		Updates:  updates.NewService(cl),
		Self:     self.NewService(cl),
	}
}
