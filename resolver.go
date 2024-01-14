package mockhttp

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/William9923/go-mockhttp/parser"
	"github.com/William9923/go-mockhttp/pathregex"
	"gopkg.in/yaml.v2"
)

// Resolver Adapter Contract:
// 1. LoadDefinition : load mock definition spec from different datastore (file, database, etc...)
// 2. Resolve        : check request and return mock response if exist
//
// used to build any datastore adapter, as long as it able to resolve mock definition properties from http request
type ResolverAdapter interface {
	LoadDefinition(ctx context.Context) error
	Resolve(ctx context.Context, req *Request) (*http.Response, error)
}

// File Based Resolver Adapter
// Use file (.yaml) based mock definition spec to resolve the mock.
type fileBasedResolver struct {
	dir         string
	definitions []fileBasedMockDefinition
	isLoaded    atomic.Bool
	template    *template.Template
}

// NewFileResolverAdapter returns new ResolverAdapter for Mock client,
// with file based mock definition.
//
// param: dir (string) -> directory path where all the mock definition specs located.
func NewFileResolverAdapter(dir string) (ResolverAdapter, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, err
	}
	return &fileBasedResolver{
		dir:         dir,
		definitions: []fileBasedMockDefinition{},
		template:    template.New("mock-svc"),
	}, nil
}

// fileBasedResolver LoadDefinition use dir field to search all the mock definition specs file (.yaml)
// and register the definitions into the adapter resolver.
//
// Also, compile all deferred field from the definitions file spec
func (r *fileBasedResolver) LoadDefinition(ctx context.Context) error {
	if r.isLoaded.Load() {
		return ErrDefinitionLoaded
	}

	fileItems, err := os.ReadDir(r.dir)
	if err != nil {
		return err
	}

	for _, item := range fileItems {
		if item.IsDir() {
			continue
		}

		f, err := os.ReadFile(filepath.Join(r.dir, item.Name()))
		if err != nil {
			return err
		}

		var definition fileBasedMockDefinition
		err = yaml.Unmarshal(f, &definition)
		if err != nil {
			return err
		}

		compiledRegex, params := pathregex.CompilePath(definition.Path, true, true)
		definition.compiledPath = compiledRegex.String()
		definition.params = params
		definition.containParams = len(params) > 0
		definition.containsWildcard = findWildcard(params)

		r.definitions = append(r.definitions, definition)
	}

	r.isLoaded.Store(true)
	return nil
}

// fileBasedResolver Resolve receive req object and
// find possible mock response from loaded mock definitions spec file (.yaml)
//
// Resolve process (file based) include these steps:
//  1. Extract request headers (and request body if it was PUT,PATCH,POST,DELETE)
//  2. Build incoming request data object
//  3. Find mock response via loaded mock definitions. The priorities of the mock definitions as below:
//     Exact path (ex: /var/william -> /var/william)
//     With path parameters (ex: /var/:name -> /var/william)
//     With wildcard (ex: /var/* -> /var/william)
//  4. Return nil with ErrNoMockResponse when no mock definitions found
//  5. Find the correct response defined in mock definitions (based on CEL rules).
//     Mock responses with rules will always be prioritized before mock responses with no rules (default)
//  6. Generate mock response body (support templating via Go text/template)
//
// WARN: req body must be using reuseable reader, as it will be read multiple time during extract request process
func (r *fileBasedResolver) Resolve(ctx context.Context, req *Request) (*http.Response, error) {

	var (
		err     error
		body    map[string]interface{}
		rawBody string
	)

	headers := extractHeader(req)

	if req.Body != nil {
		rawBody, err = extractRawBody(req)
		if err != nil {
			return nil, err
		}
		body, err = extractReqBody(req, headers)
		if err != nil {
			return nil, err
		}
	}

	request := incomingRequest{
		Host:        req.Host,
		Method:      req.Method,
		Endpoint:    pathregex.CleanPath(req.URL.EscapedPath()),
		Headers:     headers,
		Cookies:     extractCookies(req),
		QueryParams: extractQueryParam(req),
		Body:        body,
		RawBody:     rawBody,
	}

	mockResp, err := r.findMockResponse(&request, []mockDefinitionsStore{
		r.getAllExactPathDefinitions,
		r.getAllContainPathParamDefinitions,
		r.getAllHaveWildcardDefinitions,
	})
	if err != nil {
		return nil, err
	}
	if mockResp == nil {
		return nil, ErrNoMockResponse
	}

	return r.generateResp(&request, mockResp)
}

func (r *fileBasedResolver) findMockResponse(request *incomingRequest, definitionsFn []mockDefinitionsStore) (*mockResponse, error) {
	for _, fn := range definitionsFn {
		for _, definition := range fn(request.Host, request.Method) {
			if isMatch := pathregex.MatchPath(request.Endpoint, definition.Path); isMatch {
				params := pathregex.ExtractPathParam(request.Endpoint, definition.Path)
				request.RouteParams = params
				resp, err := r.findResponse(request, definition)
				if err != nil {
					return nil, err
				}
				return resp, nil
			}
		}
	}

	return nil, ErrNoMockResponse
}

// fileBasedResolver generateResp
// Generate http.Response object based on defined response from mock definition.
//
// Support templating via Go text/template if `enabled_template` is true
// The template will be filled with all parameters from request (cookies, headers, path param and query params)
func (r *fileBasedResolver) generateResp(request *incomingRequest, response *mockResponse) (*http.Response, error) {
	headers := response.ResponseHeaders
	statusCode := response.StatusCode
	body := response.Body

	if response.EnableTemplate {
		buf := new(bytes.Buffer)

		t := template.Must(r.template.Parse(body))
		if err := t.Execute(buf, request.collectAllParams()); err != nil {
			return nil, ErrCommon
		}
		body = buf.String()
	}

	actualHeaders := make(http.Header)
	isContentTypeSet := false
	for name, value := range headers {
		if name == "Content-Type" {
			isContentTypeSet = true
		}
		actualHeaders[name] = []string{value}
	}
	if !isContentTypeSet {
		contentType := http.DetectContentType([]byte(body))
		actualHeaders["Content-Type"] = []string{contentType}
	}

	return &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		StatusCode: statusCode,
		Header:     actualHeaders,
	}, nil
}

// --- Repository-like (datastore) function to get definition based on condition ---
type mockDefinitionsStore func(host, method string) []fileBasedMockDefinition

// fileBasedResolver getAllContainPathParamDefinitions
// Fetch all mock definitions that contain path param
// based on request Host and http method.
//
// ex:
// /v1/api/mock/:id => true (contain path param)
// /v1/api/mock/1   => false (exact path)
// /v1/api/mock/*   => false (have wildcard)
func (r *fileBasedResolver) getAllContainPathParamDefinitions(host, method string) []fileBasedMockDefinition {
	var dataToQuery = r.definitions
	dataToQuery = filter[fileBasedMockDefinition](dataToQuery, func(definition fileBasedMockDefinition) bool {
		return definition.Method == method && definition.containParams && !definition.containsWildcard
	})
	return dataToQuery
}

// fileBasedResolver getAllExactPathDefinitions
// Fetch all mock definitions with exact path
// based on request Host and http method.
//
// ex:
// /v1/api/mock/:id => false (contain path param)
// /v1/api/mock/1   => true (exact path)
// /v1/api/mock/*   => false (have wildcard)
func (r *fileBasedResolver) getAllExactPathDefinitions(host, method string) []fileBasedMockDefinition {
	var dataToQuery = r.definitions
	dataToQuery = filter[fileBasedMockDefinition](dataToQuery, func(definition fileBasedMockDefinition) bool {
		return definition.Method == method && definition.Host == host && !definition.containParams && !definition.containsWildcard
	})
	return dataToQuery
}

// fileBasedResolver getAllHaveWildcardDefinitions
// Fetch all mock definitions that have wildcard
// based on request Host and http method.
//
// ex:
// /v1/api/mock/:id => false (contain path param)
// /v1/api/mock/1   => false (exact path)
// /v1/api/mock/*   => true (have wildcard)
func (r *fileBasedResolver) getAllHaveWildcardDefinitions(host, method string) []fileBasedMockDefinition {
	var dataToQuery = r.definitions
	dataToQuery = filter[fileBasedMockDefinition](dataToQuery, func(definition fileBasedMockDefinition) bool {
		return definition.Method == method && definition.Host == host && definition.containParams && definition.containsWildcard
	})
	return dataToQuery
}

func findWildcard(params []string) bool {
	for _, param := range params {
		if param == "*" {
			return true
		}
	}
	return false
}

// --- Utility for extracting info from HTTP request ---
func extractHeader(req *Request) params {
	headers := make(params)
	for name, values := range req.Header {
		headers[name] = values[len(values)-1] // always take the last header value
	}
	return headers
}

func extractCookies(req *Request) params {
	cookies := make(params)
	for _, cookie := range req.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	return cookies
}

func extractQueryParam(req *Request) params {
	queryParams := make(params)
	for name, values := range req.URL.Query() {
		queryParams[name] = values[len(values)-1] // always take the last query param value
	}
	return queryParams
}

func extractRawBody(req *Request) (string, error) {
	// Read the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}

	// Convert the body to a string
	bodyString := string(body)
	return bodyString, nil
}

func extractFormReqBody(req *Request) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	err := req.ParseForm()
	if err != nil {
		return data, err
	}

	for name, values := range req.Form {
		data[name] = values[len(values)-1]
	}

	return data, nil
}

func extractReqBody(req *Request, headers params) (map[string]interface{}, error) {

	contentType, exist := headers["Content-Type"]
	if !exist {
		return make(map[string]interface{}), ErrUnsupportedContentType
	}

	checker := func(supportedContentType string) bool {
		return supportedContentType == contentType
	}

	if some(parsedFormBodyMimeTypes, checker) {
		return extractFormReqBody(req)
	}

	rawBody, err := extractRawBody(req)
	if err != nil {
		return make(map[string]interface{}), err
	}
	if some(parsedJSONBodyMimeTypes, checker) {
		return parser.ParseJSON(rawBody)
	}

	if some(parsedXMLBodyMimeTypes, checker) {
		return parser.ParseXML(rawBody)
	}

	return make(map[string]interface{}), ErrUnsupportedContentType
}
