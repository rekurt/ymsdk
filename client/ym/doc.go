// Package ym provides the core Yandex Messenger Bot API client and shared types.
//
// Create a client with [NewClient] or [NewClientWithHTTP], then pass it to
// service constructors (messages, chats, polls, etc.) or use the convenience
// aggregator in the parent "client" package.
//
//	cl := ym.NewClient(ym.Config{
//	    Token: os.Getenv("YM_TOKEN"),
//	    ErrorHandling: ymerrors.ErrorHandlingConfig{
//	        RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 3, RetryNetwork: true},
//	        RateLimitHandling: ymerrors.RateLimitHandling{UseRetryAfter: true},
//	    },
//	})
//	msgSvc := messages.NewService(cl)
//
// Full reference: https://pkg.go.dev/github.com/rekurt/ymsdk/client/ym
package ym
