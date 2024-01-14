package mockhttp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// ReaderFunc is the type of function that can be given natively to NewRequest
type ReaderFunc func() (io.Reader, error)

// ResponseHandlerFunc is a type of function that takes in a Response, and does something with it.
// The ResponseHandlerFunc is called when the HTTP client successfully receives a response and the
// The response body is not automatically closed. It must be closed either by the ResponseHandlerFunc or
// by the caller out-of-band. Failure to do so will result in a memory leak.
//
// Main purposes: to enable delay for mocking http calls
type ResponseHandlerFunc func(*http.Response) error

// LenReader is an interface implemented by many in-memory io.Reader's. Used
// for automatically sending the right Content-Length header when possible.
type LenReader interface {
	Len() int
}

// Request wraps the metadata needed to create HTTP requests.
type Request struct {
	// body is a seekable reader over the request body payload. This is
	// used to rewind the request data in between retries.
	body ReaderFunc

	responseHandler ResponseHandlerFunc

	// Embed an HTTP request directly. This makes a *Request act exactly
	// like an *http.Request so that all meta methods are supported.
	*http.Request
}

// WithContext returns wrapped Request with a shallow copy of underlying *http.Request
// with its context changed to ctx. The provided ctx must be non-nil.
func (r *Request) WithContext(ctx context.Context) *Request {
	return &Request{
		body:            r.body,
		responseHandler: r.responseHandler,
		Request:         r.Request.WithContext(ctx),
	}
}

// SetResponseHandler allows setting the response handler.
func (r *Request) SetResponseHandler(fn ResponseHandlerFunc) {
	r.responseHandler = fn
}

// BodyBytes allows accessing the request body. It is an analogue to
// http.Request's Body variable, but it returns a copy of the underlying data
// rather than consuming it.
//
// This function is not thread-safe; do not call it at the same time as another
// call, or at the same time this request is being used with Client.Do.
func (r *Request) BodyBytes() ([]byte, error) {
	if r.body == nil {
		return nil, nil
	}
	body, err := r.body()
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(body)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SetBody allows setting the request body.
//
// It is useful if a new body needs to be set without constructing a new Request.
func (r *Request) SetBody(rawBody interface{}) error {
	bodyReader, contentLength, err := getBodyReaderAndContentLength(rawBody)
	if err != nil {
		return err
	}
	r.body = bodyReader
	r.ContentLength = contentLength
	if bodyReader != nil {
		r.GetBody = func() (io.ReadCloser, error) {
			body, err := bodyReader()
			if err != nil {
				return nil, err
			}
			if rc, ok := body.(io.ReadCloser); ok {
				return rc, nil
			}
			return io.NopCloser(body), nil
		}
	} else {
		r.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
	}
	return nil
}

// WriteTo allows copying the request body into a writer.
//
// It writes data to w until there's no more data to write or
// when an error occurs. The return int64 value is the number of bytes
// written. Any error encountered during the write is also returned.
// The signature matches io.WriterTo interface.
func (r *Request) WriteTo(w io.Writer) (int64, error) {
	body, err := r.body()
	if err != nil {
		return 0, err
	}
	if c, ok := body.(io.Closer); ok {
		defer c.Close()
	}
	return io.Copy(w, body)
}

func getBodyReaderAndContentLength(rawBody interface{}) (ReaderFunc, int64, error) {
	var bodyReader ReaderFunc
	var contentLength int64

	switch body := rawBody.(type) {
	// If they gave us a function already, great! Use it.
	case ReaderFunc:
		bodyReader = body
		tmp, err := body()
		if err != nil {
			return nil, 0, err
		}
		if lr, ok := tmp.(LenReader); ok {
			contentLength = int64(lr.Len())
		}
		if c, ok := tmp.(io.Closer); ok {
			c.Close()
		}

	case func() (io.Reader, error):
		bodyReader = body
		tmp, err := body()
		if err != nil {
			return nil, 0, err
		}
		if lr, ok := tmp.(LenReader); ok {
			contentLength = int64(lr.Len())
		}
		if c, ok := tmp.(io.Closer); ok {
			c.Close()
		}

	// If a regular byte slice, we can read it over and over via new
	// readers
	case []byte:
		buf := body
		bodyReader = func() (io.Reader, error) {
			return bytes.NewReader(buf), nil
		}
		contentLength = int64(len(buf))

	// If a bytes.Buffer we can read the underlying byte slice over and
	// over
	case *bytes.Buffer:
		buf := body
		bodyReader = func() (io.Reader, error) {
			return bytes.NewReader(buf.Bytes()), nil
		}
		contentLength = int64(buf.Len())

	// We prioritize *bytes.Reader here because we don't really want to
	// deal with it seeking so want it to match here instead of the
	// io.ReadSeeker case.
	case *bytes.Reader:
		buf, err := io.ReadAll(body)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = func() (io.Reader, error) {
			return bytes.NewReader(buf), nil
		}
		contentLength = int64(len(buf))

	// Compat case
	case io.ReadSeeker:
		raw := body
		bodyReader = func() (io.Reader, error) {
			_, err := raw.Seek(0, 0)
			return io.NopCloser(raw), err
		}
		if lr, ok := raw.(LenReader); ok {
			contentLength = int64(lr.Len())
		}

	// Read all in so we can reset
	case io.Reader:
		buf, err := io.ReadAll(body)
		if err != nil {
			return nil, 0, err
		}
		if len(buf) == 0 {
			bodyReader = func() (io.Reader, error) {
				return http.NoBody, nil
			}
			contentLength = 0
		} else {
			bodyReader = func() (io.Reader, error) {
				return bytes.NewReader(buf), nil
			}
			contentLength = int64(len(buf))
		}

	// No body provided, nothing to do
	case nil:

	// Unrecognized type
	default:
		return nil, 0, fmt.Errorf("cannot handle type %T", rawBody)
	}
	return bodyReader, contentLength, nil
}

// FromRequest wraps an http.Request in a retryablehttp.Request
func FromRequest(r *http.Request) (*Request, error) {
	if r.Body != nil {
		bodyReader, _, err := getBodyReaderAndContentLength(r.Body)
		if err != nil {
			return nil, err
		}
		reader, err := bodyReader()
		if err != nil {
			return nil, err
		}
		reuseableReader := ReusableReader(reader)
		return &Request{body: func() (io.Reader, error) {
			return reuseableReader, nil
		}, Request: r}, nil
	} else {
		return &Request{Request: r}, nil
	}
}

// NewRequest creates a new wrapped request.
func NewRequest(method, url string, rawBody interface{}) (*Request, error) {
	return NewRequestWithContext(context.Background(), method, url, rawBody)
}

// NewRequestWithContext creates a new wrapped request with the provided context.
//
// The context controls the entire lifetime of a request and its response:
// obtaining a connection, sending the request, and reading the response headers and body.
func NewRequestWithContext(ctx context.Context, method, url string, rawBody interface{}) (*Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req := &Request{
		Request: httpReq,
	}
	if err := req.SetBody(rawBody); err != nil {
		return nil, err
	}

	return req, nil
}
