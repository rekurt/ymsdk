// Package ymsdk is a lightweight Go SDK for the Yandex Messenger Bot API.
//
// The SDK provides type-safe models, automatic retry with exponential back-off,
// rate-limit handling, and service-oriented architecture covering all core API
// methods: messages, chats, polls, updates, users, files, and bot self-management.
//
// Quick start — use the convenience aggregator:
//
//	import "github.com/rekurt/ymsdk/client"
//
//	cs := client.New(ym.Config{Token: "..."})
//	msg, _ := cs.Messages.SendToChat(ctx, "chat-id", "hello", nil)
//
// Or construct individual services from a core client:
//
//	cl := ym.NewClient(ym.Config{Token: "..."})
//	msgSvc := messages.NewService(cl)
//	pollSvc := polls.NewService(cl)
//
// See sub-packages for detailed documentation:
//   - [github.com/rekurt/ymsdk/client] — YMClient aggregator
//   - [github.com/rekurt/ymsdk/client/ym] — core Client and shared types
//   - [github.com/rekurt/ymsdk/client/ym/ymerrors] — error types and configuration
//   - [github.com/rekurt/ymsdk/middleware] — zap-based logging utilities
package ymsdk
