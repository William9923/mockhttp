package mockhttp

import "context"

// CheckMock specifies a policy for handling mocks. It is called
// before performing http call by the http.Client.
// If CheckMock returns true, the Client stops performing actual http call
// and returns the response from mock datastore.
// If CheckMock returns an error, the Client will perform usual http call.
// The Client will close any response body when using / not using mock
// (to properly close before returning)
type CheckMock func(ctx context.Context, req *Request) (bool, error)

// DefaultMockPolicy provides a default callback for Client.CheckRetry, which
// will only mock only if response exist.
func DefaultMockPolicy(ctx context.Context, req *Request) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	// TODO: create the way to validate it
	return true, nil
}
