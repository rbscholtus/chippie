package romstore

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/rbscholtus/chippie/catalogue"
	"github.com/rbscholtus/chippie/platform"
)

// memCache is a Cache in memory that counts its calls.
type memCache struct {
	mu   sync.Mutex
	data map[string][]byte
	puts int
	fail bool
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

	c.puts++
	if c.fail {
		return platform.ErrUnavailable
	}

	c.data[key] = value

	return nil
}

var (
	rom     = []byte{0x00, 0xE0, 0x12, 0x00}
	romHash = Hash(rom)
)

// server serves the ROM at /good, a different ROM at /wrong, a ROM over
// the size limit at /big, and 404 elsewhere. It counts requests.
func server(t *testing.T) (*httptest.Server, *int) {
	t.Helper()

	var (
		mu sync.Mutex
		n  int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n++
		mu.Unlock()

		switch r.URL.Path {
		case "/good", "/hosted/x.ch8":
			_, _ = w.Write(rom)
		case "/wrong":
			_, _ = w.Write([]byte{1, 2, 3})
		case "/big":
			_, _ = w.Write(make([]byte, MaxROMSize+1))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	return srv, &n
}

func locations(urls ...string) func(string) []catalogue.Location {
	return func(string) []catalogue.Location {
		out := make([]catalogue.Location, len(urls))
		for i, u := range urls {
			out[i] = catalogue.Location{Source: u, URL: u}
		}

		return out
	}
}

func TestFindTriesNextURLOnMismatch(t *testing.T) {
	t.Parallel()

	srv, _ := server(t)
	cache := newMemCache()
	s := New(locations(srv.URL+"/missing", srv.URL+"/wrong", srv.URL+"/good"), cache, "")

	res := s.Find(t.Context(), romHash)
	if res.Err != nil || string(res.ROM) != string(rom) || res.From != srv.URL+"/good" {
		t.Fatalf("Find = %+v", res)
	}

	if _, ok, _ := cache.Get(romHash); !ok {
		t.Error("downloaded ROM not cached")
	}
}

func TestFindAllFail(t *testing.T) {
	t.Parallel()

	srv, _ := server(t)
	s := New(locations(srv.URL+"/wrong", srv.URL+"/big", srv.URL+"/missing"), newMemCache(), "")

	res := s.Find(t.Context(), romHash)

	wrapsAll := errors.Is(res.Err, ErrUnavailable) && errors.Is(res.Err, ErrHashMismatch) &&
		errors.Is(res.Err, platform.ErrTooLarge)
	if !wrapsAll {
		t.Errorf("err=%v, want unavailable, wrapping the mismatch and the size error", res.Err)
	}

	if res := New(locations(), newMemCache(), "").Find(t.Context(), romHash); !errors.Is(res.Err, ErrUnavailable) {
		t.Errorf("no locations: err=%v", res.Err)
	}
}

func TestFindUsesCacheWithoutNetwork(t *testing.T) {
	t.Parallel()

	srv, requests := server(t)
	cache := newMemCache()
	_ = cache.Put(romHash, rom)

	res := New(locations(srv.URL+"/good"), cache, "").Find(t.Context(), romHash)
	if res.Err != nil || res.From != FromCache || *requests != 0 {
		t.Errorf("cached ROM: %+v, %d requests", res, *requests)
	}

	// A corrupt cache entry is ignored and replaced.
	cache.data[romHash] = []byte("corrupt")

	res = New(locations(srv.URL+"/good"), cache, "").Find(t.Context(), romHash)
	if res.Err != nil || res.From == FromCache || string(cache.data[romHash]) != string(rom) {
		t.Errorf("corrupt cache: %+v", res)
	}
}

func TestFindSurvivesCacheFailure(t *testing.T) {
	t.Parallel()

	srv, _ := server(t)
	cache := newMemCache()
	cache.fail = true

	if res := New(locations(srv.URL+"/good"), cache, "").Find(t.Context(), romHash); res.Err != nil {
		t.Errorf("a failing cache should not fail the download: %v", res.Err)
	}
}

func TestFindCancelled(t *testing.T) {
	t.Parallel()

	srv, requests := server(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res := New(locations(srv.URL+"/good"), newMemCache(), "").Find(ctx, romHash)
	if !errors.Is(res.Err, context.Canceled) || *requests != 0 {
		t.Errorf("cancelled: err=%v, %d requests", res.Err, *requests)
	}
}

func TestRelativeURLsUseBase(t *testing.T) {
	t.Parallel()

	srv, _ := server(t)

	res := New(locations("hosted/x.ch8"), newMemCache(), srv.URL+"/").Find(t.Context(), romHash)
	if res.Err != nil {
		t.Errorf("hosted ROM: %v", res.Err)
	}
}

func TestGetIsAsync(t *testing.T) {
	t.Parallel()

	srv, _ := server(t)

	res := <-New(locations(srv.URL+"/good"), newMemCache(), "").Get(t.Context(), romHash)
	if res.Err != nil || res.SHA1 != romHash {
		t.Errorf("Get: %+v", res)
	}
}

func TestLocalComesBeforeDownload(t *testing.T) {
	t.Parallel()

	srv, requests := server(t)
	local := func(_ context.Context, sha1 string) ([]byte, bool) { return rom, sha1 == romHash }

	res := New(locations(srv.URL+"/good"), newMemCache(), "").WithLocal(local).Find(t.Context(), romHash)
	if res.Err != nil || res.From != FromLocal || *requests != 0 {
		t.Errorf("local ROM: %+v, %d requests", res, *requests)
	}

	// A local copy that doesn't match its hash is ignored.
	wrong := func(context.Context, string) ([]byte, bool) { return []byte("nope"), true }

	res = New(locations(srv.URL+"/good"), newMemCache(), "").WithLocal(wrong).Find(t.Context(), romHash)
	if res.Err != nil || res.From == FromLocal {
		t.Errorf("bad local copy: %+v", res)
	}
}

func TestWithFetch(t *testing.T) {
	t.Parallel()

	calls := 0
	s := New(locations("x"), newMemCache(), "").WithFetch(func(context.Context, string, int) ([]byte, error) {
		calls++

		return rom, nil
	})

	if res := s.Find(t.Context(), romHash); res.Err != nil || calls != 1 {
		t.Errorf("custom fetch: %+v, %d calls", res, calls)
	}
}
