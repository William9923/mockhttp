package mockhttp

import (
	"fmt"
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

var parsedBodyMimeTypes = append(append(parsedJSONBodyMimeTypes, parsedXMLBodyMimeTypes...), parsedFormBodyMimeTypes...)

func (r *fileBasedResolver) validateTarget(req *incomingRequest) error {

	if !in[string](req.Method, []string{http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete}) {
		return nil
	}

	headers := req.Headers
	contentType, exist := headers["Content-Type"]
	if !exist {
		return fmt.Errorf("unable to find content type")
	}

	if !some[string](parsedBodyMimeTypes, func(supportedContentType string) bool {
		return supportedContentType == contentType
	}) {
		return fmt.Errorf("unsupported request content type")
	}

	return nil
}

func (r *fileBasedResolver) findResponse(request *incomingRequest, selectedPolicy FileBasedMockPolicy) (*mockResponse, error) {

	if err := r.validateTarget(request); err != nil {
		return nil, err
	}
	return r.chooseResponse(request, selectedPolicy), nil
}

func (r *fileBasedResolver) chooseResponse(request *incomingRequest, policy FileBasedMockPolicy) *mockResponse {

	fmt.Println("hid choose response")
	correctResponse, _ := find[mockResponse](policy.Responses, func(data mockResponse) bool {
		// lower the priotization of non-rules affected response
		if data.isDefault() {
			return false
		}

		return satisfyEvery[string](data.Rules, func(rule string) bool {
			return r.isRuleFulfilled(request, rule)
		})
	})
	if !correctResponse.isNil() {
		return &correctResponse
	}

	defaultResponse, _ := find[mockResponse](policy.Responses, func(data mockResponse) bool {
		return data.isDefault()
	})
	if !defaultResponse.isNil() {
		return &defaultResponse
	}

	return nil
}

// NOTE: cookie, header, query param, route param, body
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
