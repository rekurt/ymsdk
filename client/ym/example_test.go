package ym_test

import (
	"fmt"
	"os"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func ExampleNewClient() {
	cl := ym.NewClient(ym.Config{
		Token: os.Getenv("YM_TOKEN"),
	})

	fmt.Println(cl != nil)
	// Output:
	// true
}

func ExampleNewClient_withRetry() {
	cl := ym.NewClient(ym.Config{
		Token: os.Getenv("YM_TOKEN"),
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    3,
				InitialBackoff: 500 * time.Millisecond,
				MaxBackoff:     10 * time.Second,
				RetryNetwork:   true,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{
				UseRetryAfter: true,
			},
		},
	})

	fmt.Println(cl != nil)
	// Output:
	// true
}
