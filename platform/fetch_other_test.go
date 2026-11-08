//go:build !(js && wasm)

package platform

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte("hello"))
		case "/big":
			_, _ = w.Write(make([]byte, 11))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	body, err := Fetch(t.Context(), srv.URL+"/ok", 10)
	if err != nil || string(body) != "hello" {
		t.Errorf("ok: %q, %v", body, err)
	}

	if _, err := Fetch(t.Context(), srv.URL+"/big", 10); !errors.Is(err, ErrTooLarge) {
		t.Errorf("big: err=%v, want ErrTooLarge", err)
	}

	if _, err := Fetch(t.Context(), srv.URL+"/missing", 10); !errors.Is(err, ErrHTTPStatus) {
		t.Errorf("missing: err=%v, want ErrHTTPStatus", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := Fetch(ctx, srv.URL+"/ok", 10); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: err=%v, want context.Canceled", err)
	}

	if _, err := Fetch(t.Context(), "://bad", 10); err == nil {
		t.Error("bad URL: no error")
	}
}
