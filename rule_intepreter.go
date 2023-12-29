package mockhttp

import (
	"context"
	"net/http"
)

type Resolver interface {
	Resolve(ctx context.Context, req *Request) (*http.Response, error)
}

type standardResolver struct{}

func (r standardResolver) Resolve(ctx context.Context, req *Request) (*http.Response, error) {
	return &http.Response{
		Body:       nil,
		StatusCode: http.StatusOK,
	}, nil
}
