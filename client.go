// standard net/http client library and exposes nearly the same public API.
// This makes mocking http response very easy to drop into existing programs.
//
// mockhttp performs mock if enabled. Inspired by retryablehttp by hashicorp.
//
// As noted from retryablehttp documentation:
// Requests which take a request body should provide a non-nil function
// parameter. The best choice is to provide either a function satisfying
// ReaderFunc which provides multiple io.Readers in an efficient manner, a
// *bytes.Buffer (the underlying raw byte slice will be used) or a raw byte
// slice. As it is a reference type, and we will wrap it as needed by readers,
// we can efficiently re-use the request body without needing to copy it. If an
// io.Reader (such as a *bytes.Reader) is provided, the full body will be read
// prior to the first request, and will be efficiently re-used for any retries.
// ReadSeeker can be used, but some users have observed occasional data races
// between the net/http library and the Seek functionality of some
// implementations of ReadSeeker, so should be avoided if possible.

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
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

var (
	// Default mock configuration

	// defaultLogger is the logger provided with defaultClient
	defaultLogger = log.New(os.Stderr, "", log.LstdFlags)

	// defaultResolver is the req => resp mock resolver
	defaultResolver = standardResolver{} // TODO: change implementation

	// defaultClient is used for performing requests without explicitly making
	// a new client. It is purposely private to avoid modifications.
	defaultClient = NewClient()

	// We need to consume response bodies to maintain http connections, but
	// limit the size we consume to respReadLimit.
	respReadLimit = int64(4096)

	// defaultDelay is used for mimicing actual http latency.
	// use milliseconds as unit of measurement.
	defaultDelay = 1 * time.Millisecond
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

	// CheckRetry specifies the policy for handling mock, and is called
	// befeore each request. The default policy is DefaultMockPolicy.
	CheckMock CheckMock

	// MockStore represents the datastore.
	// The built-in library provides file-based datastore, but it can be easily extended.
	Resolver Resolver

	// Delay specifies the delay if mock http call occurs. Default is 0ms
	Delay time.Duration

	loggerInit sync.Once
	clientInit sync.Once
}

// NewClient creates a new Client with default settings.
func NewClient() *Client {
	return &Client{
		HTTPClient: cleanhttp.DefaultPooledClient(),
		Logger:     defaultLogger,
		CheckMock:  DefaultMockPolicy,
		Delay:      defaultDelay,
		Resolver:   defaultResolver,
	}
}

// WithResolver returns client with custom resolver to finding the mock
func (c *Client) WithResolver(resolver Resolver) *Client {
	c.Resolver = resolver
	return c
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

	startTime := time.Now()

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
	var shouldMock bool

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
	shouldMock, _ = c.CheckMock(req.Context(), req)
	if shouldMock {
		time.Sleep(c.Delay - time.Since(startTime))
		return c.Resolver.Resolve(req.Context(), req)
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

	time.Sleep(c.Delay - time.Since(startTime))
	return resp, err
}

// Get is a shortcut for doing a GET request without making a new client.
func Get(url string) (*http.Response, error) {
	return defaultClient.Get(url)
}

// Get is a convenience helper for doing simple GET requests.
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Head is a shortcut for doing a HEAD request without making a new client.
func Head(url string) (*http.Response, error) {
	return defaultClient.Head(url)
}

// Head is a convenience method for doing simple HEAD requests.
func (c *Client) Head(url string) (*http.Response, error) {
	req, err := NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post is a shortcut for doing a POST request without making a new client.
func Post(url, bodyType string, body interface{}) (*http.Response, error) {
	return defaultClient.Post(url, bodyType, body)
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

// PostForm is a shortcut to perform a POST with form data without creating
// a new client.
func PostForm(url string, data url.Values) (*http.Response, error) {
	return defaultClient.PostForm(url, data)
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
		Transport: &RoundTripper{Client: c},
	}
}
