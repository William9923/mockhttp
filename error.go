package mockhttp

import "fmt"

var (
	ErrPolicyLoaded  = fmt.Errorf("policy loaded")
	ErrClientMissing = fmt.Errorf("client missing")
)
