package ymerrors_test

import (
	"errors"
	"fmt"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func ExampleAPIError() {
	err := &ymerrors.APIError{
		Kind:        ymerrors.KindRateLimited,
		HTTPStatus:  429,
		Description: "Too many requests",
	}

	fmt.Println(err.Error())
	fmt.Println(errors.Is(err, ymerrors.ErrRateLimited))
	// Output:
	// yandex-messenger/apierror: kind=1 http=429: Too many requests
	// true
}

func ExampleAPIError_unwrap() {
	apiErr := &ymerrors.APIError{
		Kind:        ymerrors.KindInvalidToken,
		HTTPStatus:  403,
		Description: "Invalid token",
	}

	// Use errors.As to extract the structured error.
	var target *ymerrors.APIError
	if errors.As(apiErr, &target) {
		fmt.Println("HTTP status:", target.HTTPStatus)
		fmt.Println("Description:", target.Description)
	}
	// Output:
	// HTTP status: 403
	// Description: Invalid token
}
