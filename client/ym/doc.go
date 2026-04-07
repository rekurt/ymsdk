// Package ym provides the core Yandex Messenger Bot API client and shared types.
//
// Create a client with [NewClient] or [NewClientWithHTTP], then pass it to
// service constructors (messages, chats, polls, etc.) or use the convenience
// aggregator in the parent "client" package.
//
// Using the aggregator (recommended for most use cases):
//
//	cs := client.New(ym.Config{
//	    Token: os.Getenv("YM_TOKEN"),
//	    ErrorHandling: ymerrors.ErrorHandlingConfig{
//	        RetryStrategy:     ymerrors.RetryStrategy{MaxAttempts: 3, RetryNetwork: true},
//	        RateLimitHandling: ymerrors.RateLimitHandling{UseRetryAfter: true},
//	    },
//	})
//	msg, _ := cs.Messages.SendToChat(ctx, "chat-id", "hello", nil)
//
// Using individual services:
//
//	cl := ym.NewClient(ym.Config{Token: os.Getenv("YM_TOKEN")})
//	msgSvc := messages.NewService(cl)
//	pollSvc := polls.NewService(cl)
//
// Shared types ([ChatID], [UserLogin], [MessageID], [Update], [Message], etc.)
// are defined in this package and used across all services.
//
// Use [Update.ToMessage] to convert an incoming update to a [Message] struct.
//
// Full reference: https://pkg.go.dev/github.com/rekurt/ymsdk/client/ym
package ym
