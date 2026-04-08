// Package messages provides operations for sending, deleting, and managing
// messages in Yandex Messenger chats.
//
// The service supports plain-text messages, file and image attachments,
// image galleries, and message deletion. Messages can be sent to a chat
// by [ym.ChatID] or directly to a user by [ym.UserLogin].
//
// Create the service via [NewService] with a configured [ym.Client]:
//
//	svc := messages.NewService(ymClient)
//	msg, err := svc.SendToChat(ctx, chatID, "Hello!", nil)
//
// File operations include uploading files and images, sending multi-image
// galleries, and downloading files by ID.
package messages
