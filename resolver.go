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

type ResolverAdapter interface {
	LoadPolicy(ctx context.Context) error
	Resolve(ctx context.Context, req *Request) (*http.Response, error)
}

// --- File Based Adapter ---
type fileBasedResolver struct {
	dir      string
	policies []FileBasedMockPolicy
	isLoaded atomic.Bool
	template *template.Template
}

func NewFileResolverAdapter(dir string) (ResolverAdapter, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, err
	}
	return &fileBasedResolver{
		dir:      dir,
		policies: []FileBasedMockPolicy{},
		template: template.New("mock-svc"),
	}, nil
}

// --- Model ---
type FileBasedMockPolicy struct {
	Host      string         `yaml:"host"`
	Path      string         `yaml:"path"`
	Method    string         `yaml:"method"`
	Desc      string         `yaml:"desc"`
	Responses []mockResponse `yaml:"responses"`

	// deferred field
	compiledPath     string
	params           []string
	containParams    bool
	containsWildcard bool
}

type mockResponse struct {
	ResponseHeaders map[string]string `yaml:"response_headers"`
	Rules           []string          `yaml:"rules"`
	Delay           int               `yaml:"delay"`
	StatusCode      int               `yaml:"status_code"`
	EnableTemplate  bool              `yaml:"enable_template"`
	Body            string            `yaml:"response_body"`
}

func (r *mockResponse) isNil() bool {
	return r.StatusCode == 0 && r.Body == "" && len(r.Rules) == 0
}

func (r *mockResponse) isDefault() bool {
	return len(r.Rules) == 0
}

type params map[string]string

func (p params) export() map[string]interface{} {
	interfaceMap := make(map[string]interface{})

	for key, value := range p {
		interfaceMap[key] = value
	}

	return interfaceMap
}

type incomingRequest struct {
	Host        string
	Method      string
	Endpoint    string
	Headers     params
	Cookies     params
	QueryParams params
	RouteParams params
	Body        map[string]interface{}
	RawBody     string
}

func (req incomingRequest) collectAllParams() params {
	return mergeMaps([]params{req.QueryParams, req.Cookies, req.Headers, req.RouteParams})
}

func mergeMaps(data []params) params {
	merged := make(params)
	for _, param := range data {
		for key, value := range param {
			merged[key] = value
		}
	}
	return merged
}

// --- LOAD ---

func (r *fileBasedResolver) LoadPolicy(ctx context.Context) error {
	if r.isLoaded.Load() {
		return ErrPolicyLoaded
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

		var policy FileBasedMockPolicy
		err = yaml.Unmarshal(f, &policy)
		if err != nil {
			return err
		}

		compiledRegex, params := pathregex.CompilePath(policy.Path, true, true)
		policy.compiledPath = compiledRegex.String()
		policy.params = params
		policy.containParams = len(params) > 0
		policy.containsWildcard = findWildcard(params)

		r.policies = append(r.policies, policy)
	}

	r.isLoaded.Store(true)
	return nil
}

// --- Get Policies Functions ---
func filterByCondition(collections []FileBasedMockPolicy, condition func(FileBasedMockPolicy) bool) []FileBasedMockPolicy {
	var result []FileBasedMockPolicy
	for _, data := range collections {
		if condition(data) {
			result = append(result, data)
		}
	}

	return result
}

type policyStore func(host, method string) []FileBasedMockPolicy

func (r *fileBasedResolver) getAllContainParamRoutes(host, method string) []FileBasedMockPolicy {
	var dataToQuery = r.policies
	dataToQuery = filterByCondition(dataToQuery, func(policy FileBasedMockPolicy) bool {
		return policy.Method == method && policy.containParams && !policy.containsWildcard
	})
	return dataToQuery
}

func (r *fileBasedResolver) getAllExactEndpointRoutes(host, method string) []FileBasedMockPolicy {
	var dataToQuery = r.policies
	dataToQuery = filterByCondition(dataToQuery, func(policy FileBasedMockPolicy) bool {
		return policy.Method == method && policy.Host == host && !policy.containParams && !policy.containsWildcard
	})
	return dataToQuery
}

func (r *fileBasedResolver) getAllWildcardRoutes(host, method string) []FileBasedMockPolicy {
	var dataToQuery = r.policies
	dataToQuery = filterByCondition(dataToQuery, func(policy FileBasedMockPolicy) bool {
		return policy.Method == method && policy.Host == host && policy.containParams && policy.containsWildcard
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

/*
	    Referenced from net/http library
		     Method         = "OPTIONS"                ; Section 9.2
		                    | "GET"                    ; Section 9.3
		                    | "HEAD"                   ; Section 9.4
		                    | "POST"                   ; Section 9.5
		                    | "PUT"                    ; Section 9.6
		                    | "DELETE"                 ; Section 9.7
		                    | "TRACE"                  ; Section 9.8
		                    | "CONNECT"                ; Section 9.9
		                    | extension-method
		   extension-method = token
		     token          = 1*<any CHAR except CTLs or separators>
*/
func (r *fileBasedResolver) Resolve(ctx context.Context, req *Request) (*http.Response, error) {

	var (
		err     error
		body    map[string]interface{}
		rawBody string
	)

	headers := extractHeader(req)

	if in[string](req.Method, []string{http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete}) {
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

	mockResp, err := r.findMockResponse(&request, []policyStore{
		r.getAllExactEndpointRoutes,
		r.getAllContainParamRoutes,
		r.getAllWildcardRoutes,
	})
	if err != nil {
		return nil, err
	}
	if mockResp == nil {
		return nil, ErrNoMockResponse
	}

	return r.generateResp(&request, mockResp)
}

func (r *fileBasedResolver) findMockResponse(request *incomingRequest, policiesFn []policyStore) (*mockResponse, error) {
	for _, fn := range policiesFn {
		for _, policy := range fn(request.Host, request.Method) {
			if isMatch := pathregex.MatchPath(request.Endpoint, policy.Path); isMatch {
				params := pathregex.ExtractPathParam(request.Endpoint, policy.Path)
				request.RouteParams = params
				resp, err := r.findResponse(request, policy)
				if err != nil {
					return nil, err
				}
				return resp, nil
			}
		}
	}

	return nil, ErrNoMockResponse
}

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

	if satisfyAtLeastOne(parsedFormBodyMimeTypes, checker) {
		return extractFormReqBody(req)
	}

	rawBody, err := extractRawBody(req)
	if err != nil {
		return make(map[string]interface{}), err
	}
	if satisfyAtLeastOne(parsedJSONBodyMimeTypes, checker) {
		return parser.ParseJSON(rawBody)
	}

	if satisfyAtLeastOne(parsedXMLBodyMimeTypes, checker) {
		return parser.ParseXML(rawBody)
	}

	return make(map[string]interface{}), ErrUnsupportedContentType
}
