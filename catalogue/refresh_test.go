package catalogue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rbscholtus/chippie/platform"
)

type memCache struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMemCache() *memCache { return &memCache{data: map[string][]byte{}} }

func (c *memCache) Get(key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, ok := c.data[key]

	return v, ok, nil
}

func (c *memCache) Put(key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value

	return nil
}

// dbServer serves the snapshot's files, with one file replaced by body or,
// if status isn't 200, failing with that status.
func dbServer(t *testing.T, replace string, body []byte, status int) string {
	t.Helper()

	snap, err := SnapshotFiles()
	if err != nil {
		t.Fatal(err)
	}

	files := map[string][]byte{
		"platforms.json": snap.Platforms, "quirks.json": snap.Quirks,
		"programs.json": snap.Programs, "sha1-hashes.json": snap.Hashes,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/db/")
		if name == replace {
			if status != http.StatusOK {
				w.WriteHeader(status)

				return
			}

			_, _ = w.Write(body)

			return
		}

		_, _ = w.Write(files[name])
	}))
	t.Cleanup(srv.Close)

	return srv.URL + "/db/"
}

func TestRefreshFromNetwork(t *testing.T) {
	t.Parallel()

	cache := newMemCache()

	res := <-Refresh(t.Context(), platform.Fetch, cache, dbServer(t, "", nil, http.StatusOK))
	if res.Err != nil || res.From != FromNetwork || res.Catalogue == nil {
		t.Fatalf("refresh: %+v", res)
	}

	if len(cache.data) != 4 {
		t.Errorf("cached %d files, want 4", len(cache.data))
	}
}

func TestRefreshKeepsSnapshotOnBadData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		replace string
		body    string
		status  int
	}{
		{"bad JSON", "programs.json", "{", http.StatusOK},
		{"unknown quirk", "platforms.json", `[{"id":"x","defaultTickrate":1,"quirks":{"nope":true}}]`, http.StatusOK},
		{"broken cross-reference", "sha1-hashes.json", `{"0000":0}`, http.StatusOK},
		{"server error", "quirks.json", "", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cache := newMemCache()
			base := dbServer(t, tt.replace, []byte(tt.body), tt.status)

			res := <-Refresh(t.Context(), platform.Fetch, cache, base)
			if !errors.Is(res.Err, ErrNoRefresh) || res.Catalogue != nil {
				t.Errorf("refresh: %+v, want ErrNoRefresh", res)
			}

			if len(cache.data) != 0 {
				t.Error("invalid data was cached")
			}
		})
	}
}

func TestRefreshFallsBackToCache(t *testing.T) {
	t.Parallel()

	cache := newMemCache()
	good := dbServer(t, "", nil, http.StatusOK)

	if res := <-Refresh(t.Context(), platform.Fetch, cache, good); res.Err != nil {
		t.Fatal(res.Err)
	}

	// Offline now: the cached copy is used.
	offline := dbServer(t, "programs.json", nil, http.StatusServiceUnavailable)

	res := <-Refresh(t.Context(), platform.Fetch, cache, offline)
	if res.Err != nil || res.From != FromCache || res.Catalogue == nil {
		t.Errorf("offline refresh: %+v, want the cached copy", res)
	}

	// A corrupt cache isn't used either.
	cache.data["quirks.json"] = []byte("[")

	if res := <-Refresh(t.Context(), platform.Fetch, cache, offline); !errors.Is(res.Err, ErrNoRefresh) {
		t.Errorf("corrupt cache: %+v", res)
	}
}

func TestRefreshTimesOut(t *testing.T) {
	t.Parallel()

	slow := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	t.Cleanup(slow.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()

	res := <-Refresh(ctx, platform.Fetch, newMemCache(), slow.URL+"/")
	if !errors.Is(res.Err, ErrNoRefresh) || time.Since(start) > 2*time.Second {
		t.Errorf("slow server: %+v after %v", res, time.Since(start))
	}
}
