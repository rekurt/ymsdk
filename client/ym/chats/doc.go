// Package chats provides operations for creating and managing chats and
// channels in Yandex Messenger.
//
// The service supports creating group chats and channels, as well as
// updating chat membership (adding and removing members).
//
// Create the service via [NewService] with a configured [ym.Client]:
//
//	svc := chats.NewService(ymClient)
//	chat, err := svc.Create(ctx, &chats.ChatCreateRequest{
//	    Name:    "Team Chat",
//	    Members: []ym.ChatMember{{Login: "alice"}},
//	})
package chats
