package platform

import "errors"

// Errors from Fetch.
var (
	ErrHTTPStatus = errors.New("platform: HTTP request failed")
	ErrTooLarge   = errors.New("platform: response too large")
)

// Fetch downloads url and returns the body, at most limit bytes. It blocks,
// so call it from a goroutine, never from Update or Draw. In the browser it
// uses fetch() directly, because Go's net/http adds about 1.7 MB (gzip) to
// the WebAssembly binary; native builds use net/http.
//
// func Fetch(ctx context.Context, url string, limit int) ([]byte, error)
// is implemented in fetch_js.go and fetch_other.go.
