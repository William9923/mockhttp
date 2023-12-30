package mockhttp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// RoundTripper implements the http.RoundTripper interface, using a mock-able
// HTTP client to execute requests.
type roundTripper struct {
	Client *Client
}

// RoundTrip satisfies the http.RoundTripper interface.
func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	if rt.Client == nil {
		return nil, fmt.Errorf("empty client")
	}

	// Convert the request to be retryable.
	retryableReq, err := FromRequest(req)
	if err != nil {
		return nil, err
	}

	// Execute the request.
	resp, err := rt.Client.Do(retryableReq)
	// If we got an error returned by standard library's `Do` method, unwrap it
	// otherwise we will wind up erroneously re-nesting the error.
	if _, ok := err.(*url.Error); ok {
		return resp, errors.Unwrap(err)
	}

	return resp, err
}
