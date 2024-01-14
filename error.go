package mockhttp

import "fmt"

var (
	ErrDefinitionLoaded       = fmt.Errorf("mock definition had been loaded")
	ErrClientMissing          = fmt.Errorf("client missing")
	ErrNoMockResponse         = fmt.Errorf("no mock response prepared")
	ErrUnsupportedContentType = fmt.Errorf("unsupported content type")
	ErrCommon                 = fmt.Errorf("common error")
	ErrNoContentType          = fmt.Errorf("unable to find content type")
)
