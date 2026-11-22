//go:build js && wasm

package platform

import (
	"fmt"
	"net/url"
	"syscall/js"
)

// ReadParams reads the page's URL query (?rom=…&shader=…&debug=1). The
// arguments are ignored; they exist to match the native signature.
func ReadParams(_ []string) (Params, error) {
	loc := js.Global().Get("location")

	q, err := url.ParseQuery(trimQuestion(loc.Get("search").String()))
	if err != nil {
		return Params{}, fmt.Errorf("parse URL query: %w", err)
	}

	base, err := url.Parse(loc.Get("href").String())
	if err != nil {
		return Params{}, fmt.Errorf("parse page URL: %w", err)
	}

	tests := base.ResolveReference(&url.URL{Path: "roms/test-suite/"})
	dir := base.ResolveReference(&url.URL{Path: "./"})

	shader := q.Get("shader")
	if shader == "" {
		shader = ShaderCRT
	}

	return Params{
		ROM:       q.Get("rom"),
		Platform:  q.Get("platform"),
		Shader:    shader,
		TestROMs:  tests.String(),
		Debug:     q.Get("debug") == "1",
		Calibrate: q.Get("skin") == "debug",
		Base:      dir.String(),
	}, nil
}

func trimQuestion(s string) string {
	if len(s) > 0 && s[0] == '?' {
		return s[1:]
	}

	return s
}
