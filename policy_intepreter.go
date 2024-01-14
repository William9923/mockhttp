package mockhttp

import (
	"net/http"

	"github.com/expr-lang/expr"
)

var parsedXMLBodyMimeTypes = []string{
	"application/xml",
	"application/soap+xml",
	"text/xml",
}

var parsedFormBodyMimeTypes = []string{
	"application/x-www-form-urlencoded",
	"multipart/form-data",
}

var parsedJSONBodyMimeTypes = []string{
	"application/json",
}

var parsedBodyMimeTypes = merge(parsedXMLBodyMimeTypes, parsedJSONBodyMimeTypes, parsedFormBodyMimeTypes)

func (r *fileBasedResolver) validateTarget(req *incomingRequest) error {

	if in[string](req.Method, []string{http.MethodGet, http.MethodHead, http.MethodDelete}) {
		return nil
	}

	headers := req.Headers
	contentType, exist := headers["Content-Type"]
	if !exist {
		return ErrNoContentType
	}

	if !some[string](parsedBodyMimeTypes, func(supportedContentType string) bool {
		return supportedContentType == contentType
	}) {
		return ErrUnsupportedContentType
	}

	return nil
}

func (r *fileBasedResolver) findResponse(request *incomingRequest, selectedPolicy fileBasedMockPolicy) (*mockResponse, error) {

	if err := r.validateTarget(request); err != nil {
		return nil, err
	}
	return r.chooseResponse(request, selectedPolicy), nil
}

func (r *fileBasedResolver) chooseResponse(request *incomingRequest, policy fileBasedMockPolicy) *mockResponse {

	correctResponse, _ := findFirst[mockResponse](policy.Responses, func(data mockResponse) bool {
		// lower the priotization of non-rules / default affected response
		if data.isDefault() {
			return false
		}

		return all[string](data.Rules, func(rule string) bool {
			return r.isRuleFulfilled(request, rule)
		})
	})
	if !correctResponse.isNil() {
		return &correctResponse
	}

	// if no mock response found, can use default one response (with no rule)
	defaultResponse, _ := findFirst[mockResponse](policy.Responses, func(data mockResponse) bool {
		return data.isDefault()
	})
	if !defaultResponse.isNil() {
		return &defaultResponse
	}

	return nil
}

// TODO: change into cel implementation...
func (r *fileBasedResolver) isRuleFulfilled(request *incomingRequest, rule string) bool {
	evalRes, err := expr.Eval(rule, map[string]interface{}{
		"raw":         request.RawBody,
		"body":        request.Body,
		"routeParams": request.RouteParams.export(),
		"headers":     request.Headers.export(),
		"cookies":     request.Cookies.export(),
		"queryParams": request.QueryParams.export(),
	})
	if err != nil {
		return false
	}
	return evalRes.(bool)
}
