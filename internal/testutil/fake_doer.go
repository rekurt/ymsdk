package testutil

import "net/http"

// FakeDoer is a mock HTTP client for testing. It replays paired Responses and
// Errors slices in order: the first call returns Responses[0]/Errors[0], the
// second returns Responses[1]/Errors[1], and so on. If an index exceeds
// the length of either slice, nil is returned for that component.
// All incoming requests are recorded in the Requests slice.
type FakeDoer struct {
	Responses []*http.Response
	Errors    []error
	Requests  []*http.Request
	idx       int
}

// Do records the request and returns the next Response/Error pair.
func (f *FakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.Requests = append(f.Requests, req)
	if f.idx >= len(f.Responses) && f.idx >= len(f.Errors) {
		return nil, nil
	}

	var resp *http.Response
	if f.idx < len(f.Responses) {
		resp = f.Responses[f.idx]
	}

	var err error
	if f.idx < len(f.Errors) {
		err = f.Errors[f.idx]
	}

	f.idx++

	return resp, err
}

// Reset clears recorded requests and resets the response index to zero,
// allowing the FakeDoer to be reused across subtests.
func (f *FakeDoer) Reset() {
	f.idx = 0
	f.Requests = nil
}

// CallCount returns the number of requests that have been made.
func (f *FakeDoer) CallCount() int {
	return len(f.Requests)
}
