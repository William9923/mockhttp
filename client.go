// standard net/http client library and exposes nearly the same public API.
// This makes mocking http response very easy to drop into existing programs.
//
// mockhttp performs mock if enabled. Inspired by retryablehttp by hashicorp.
package mockhttp

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/go-cleanhttp"
)

var (
	// Default mock configuration

	// defaultLogger is the logger provided with defaultClient
	defaultLogger = log.New(os.Stderr, "", log.LstdFlags)

	// We need to consume response bodies to maintain http connections, but
	// limit the size we consume to respReadLimit.
	respReadLimit = int64(4096)
)

// Client is used to make HTTP requests. It adds additional functionality
// like automatic retries to tolerate minor outages.
type Client struct {
	HTTPClient *http.Client // Internal HTTP client.
	Logger     interface{}  // Customer logger instance. Can be either Logger or LeveledLogger

	// RequestLogHook allows a user-supplied function to be called
	// before each httprequest  call.
	RequestLogHook RequestLogHook

	// ResponseLogHook allows a user-supplied function to be called
	// with the response from each HTTP request executed.
	ResponseLogHook ResponseLogHook

	// MockStore represents the datastore.
	// The built-in library provides file-based datastore, but it can be easily extended.
	Resolver ResolverAdapter

	loggerInit sync.Once
	clientInit sync.Once
}

// NewClient creates a new Client with default settings.
func NewClient(resolver ResolverAdapter) *Client {
	return &Client{
		HTTPClient: cleanhttp.DefaultPooledClient(),
		Logger:     defaultLogger,
		Resolver:   resolver,
	}
}

func (c *Client) logger() interface{} {
	c.loggerInit.Do(func() {
		if c.Logger == nil {
			return
		}

		switch c.Logger.(type) {
		case Logger, LeveledLogger:
			// ok
		default:
			// This should happen in dev when they are setting Logger and work on code, not in prod.
			panic(fmt.Sprintf("invalid logger type passed, must be Logger or LeveledLogger, was %T", c.Logger))
		}
	})

	return c.Logger
}

// Do wraps calling an HTTP method with retries.
func (c *Client) Do(req *Request) (*http.Response, error) {
	c.clientInit.Do(func() {
		if c.HTTPClient == nil {
			c.HTTPClient = cleanhttp.DefaultPooledClient()
		}
	})

	logger := c.logger()
	if logger != nil {
		switch v := logger.(type) {
		case LeveledLogger:
			v.Debug("performing request", "method", req.Method, "url", req.URL)
		case Logger:
			v.Printf("[DEBUG] %s %s", req.Method, req.URL)
		}
	}

	var resp *http.Response
	if req.body != nil {
		body, readErr := req.body()
		if readErr != nil {
			c.HTTPClient.CloseIdleConnections()
			return resp, readErr
		}
		if c, ok := body.(io.ReadCloser); ok {
			req.Body = c
		} else {
			req.Body = io.NopCloser(body)
		}
	}

	if c.RequestLogHook != nil {
		switch v := logger.(type) {
		case LeveledLogger:
			c.RequestLogHook(hookLogger{v}, req.Request)
		case Logger:
			c.RequestLogHook(v, req.Request)
		default:
			c.RequestLogHook(nil, req.Request)
		}
	}

	// Check if we should continue with actual http call / use mock
	mockResponse, _ := c.Resolver.Resolve(req.Context(), req)
	if mockResponse != nil {
		return mockResponse, nil
	}

	// Attempt the request
	resp, err := c.HTTPClient.Do(req.Request)
	if err != nil {
		switch v := logger.(type) {
		case LeveledLogger:
			v.Error("request failed", "error", err, "method", req.Method, "url", req.URL)
		case Logger:
			v.Printf("[ERR] %s %s request failed: %v", req.Method, req.URL, err)
		}
	} else {
		// Call this here to maintain the behavior of logging all requests,
		// even if CheckRetry signals to stop.
		if c.ResponseLogHook != nil {
			// Call the response logger function if provided.
			switch v := logger.(type) {
			case LeveledLogger:
				c.ResponseLogHook(hookLogger{v}, resp)
			case Logger:
				c.ResponseLogHook(v, resp)
			default:
				c.ResponseLogHook(nil, resp)
			}
		}
	}
	defer c.HTTPClient.CloseIdleConnections()

	return resp, err
}

// Get is a convenience helper for doing simple GET requests.
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Head is a convenience method for doing simple HEAD requests.
func (c *Client) Head(url string) (*http.Response, error) {
	req, err := NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post is a convenience method for doing simple POST requests.
func (c *Client) Post(url, bodyType string, body interface{}) (*http.Response, error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(req)
}

// PostForm is a convenience method for doing simple POST operations using
// pre-filled url.Values form data.
func (c *Client) PostForm(url string, data url.Values) (*http.Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// StandardClient returns a stdlib *http.Client with a custom Transport, which
// shims in a *mockhttp.Client for added retries.
func (c *Client) StandardClient() *http.Client {
	return &http.Client{
		Transport: &roundTripper{Client: c},
	}
}
