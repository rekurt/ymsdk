// Package files provides operations for uploading files to Yandex Messenger
// chats.
//
// Files can be sent to a chat by [ym.ChatID] or directly to a user by
// [ym.UserLogin]. The service handles multipart form encoding and
// automatic retry on transient failures.
//
// Create the service via [NewService] with a configured [ym.Client]:
//
//	svc := files.NewService(ymClient)
//	msg, err := svc.SendToChat(ctx, chatID, "report.pdf", fileReader)
package files
