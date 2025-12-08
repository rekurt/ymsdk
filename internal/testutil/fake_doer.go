package testutil

import "net/http"

type FakeDoer struct {
	Responses []*http.Response
	Errors    []error
	Requests  []*http.Request
	idx       int
}

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
