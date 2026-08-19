package client

import "time"

const (
	DefaultBaseURL     = "https://api.typecast.ai"
	DefaultHTTPTimeout = 60 * time.Second
)

// Version is the Cast CLI version reported in User-Agent.
// Release builds override it through GoReleaser ldflags.
var Version = "1.0.6"
