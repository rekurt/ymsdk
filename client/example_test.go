package client_test

import (
	"fmt"
	"os"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func ExampleNew() {
	cs := client.New(ym.Config{
		Token: os.Getenv("YM_TOKEN"),
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:  3,
				RetryNetwork: true,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{
				UseRetryAfter: true,
			},
		},
	})

	fmt.Println(cs.Messages != nil)
	fmt.Println(cs.Chats != nil)
	fmt.Println(cs.Polls != nil)
	// Output:
	// true
	// true
	// true
}

func ExampleWrap() {
	cl := ym.NewClient(ym.Config{Token: "my-token"})
	cs := client.Wrap(cl)

	fmt.Println(cs.Client != nil)
	fmt.Println(cs.Updates != nil)
	// Output:
	// true
	// true
}
