package mockhttp

type fileBasedMockDefinition struct {
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
