package mockhttp

import "fmt"

var (
	ErrPolicyLoaded           = fmt.Errorf("policy loaded")
	ErrClientMissing          = fmt.Errorf("client missing")
	ErrNoMockResponse         = fmt.Errorf("no mock response prepared")
	ErrUnsupportedContentType = fmt.Errorf("unsupported content type")
	ErrCommon                 = fmt.Errorf("common error")
)
