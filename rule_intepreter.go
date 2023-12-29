package mockhttp

import (
	"context"
	"net/http"
)

// TODO: add notes
type ResolverAdapter interface {
	LoadPolicy(ctx context.Context) error
	Resolve(ctx context.Context, req *Request) (*http.Response, error)
}

// --- File Based Adapter
type fileBasedResolver struct{}

func (r fileBasedResolver) LoadPolicy(ctx context.Context) error {
	return nil
}

func (r fileBasedResolver) Resolve(ctx context.Context, req *Request) (*http.Response, error) {
	return &http.Response{
		Body:       nil,
		StatusCode: http.StatusOK,
	}, nil
}

// TODO: need to have
func DefaultResolver() ResolverAdapter {
	return &fileBasedResolver{}
}

// --- Model
type FileBasedMockPolicy struct {
}
