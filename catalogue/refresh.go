package catalogue

import (
	"context"
	"errors"
	"fmt"
)

// DatabaseURL is where the latest chip-8-database files are fetched from
// (GitHub raw, which allows cross-origin requests).
const DatabaseURL = "https://raw.githubusercontent.com/chip-8/chip-8-database/master/database/"

// maxDatabaseFile bounds one downloaded database file.
const maxDatabaseFile = 8 << 20

// FetchFunc downloads a URL, at most limit bytes (platform.Fetch).
type FetchFunc func(ctx context.Context, url string, limit int) ([]byte, error)

// Cache keeps the last good database files (platform.Cache for the
// metadata store).
type Cache interface {
	Get(key string) ([]byte, bool, error)
	Put(key string, value []byte) error
}

// Where a refreshed catalogue came from.
const (
	FromNetwork = "network"
	FromCache   = "cache"
)

// RefreshResult is a newer catalogue, or why there isn't one. On an error
// the app keeps what it has.
type RefreshResult struct {
	Catalogue *Catalogue
	From      string
	Err       error
}

// ErrNoRefresh reports that neither the network nor the cache gave a valid
// database.
var ErrNoRefresh = errors.New("catalogue: no newer database")

// Refresh fetches the latest database in a goroutine (README §10). A valid
// download is cached and delivered; if the download fails or doesn't
// validate, the last good cached copy is delivered instead; otherwise an
// error, and the app keeps its snapshot. The channel has room for the
// result, so the game loop polls it without waiting.
func Refresh(ctx context.Context, fetch FetchFunc, cache Cache, base string) <-chan RefreshResult {
	ch := make(chan RefreshResult, 1)

	go func() { ch <- refresh(ctx, fetch, cache, base) }()

	return ch
}

func refresh(ctx context.Context, fetch FetchFunc, cache Cache, base string) RefreshResult {
	files, netErr := download(ctx, fetch, base)
	if netErr == nil {
		c, err := LoadWith(files)
		if err == nil {
			store(cache, files)

			return RefreshResult{Catalogue: c, From: FromNetwork}
		}

		netErr = err
	}

	cached, ok := fromCache(cache)
	if ok {
		if c, err := LoadWith(cached); err == nil {
			return RefreshResult{Catalogue: c, From: FromCache}
		}
	}

	return RefreshResult{Err: fmt.Errorf("%w: %w", ErrNoRefresh, netErr)}
}

func download(ctx context.Context, fetch FetchFunc, base string) (Files, error) {
	var f Files

	for _, part := range f.parts() {
		b, err := fetch(ctx, base+part.name, maxDatabaseFile)
		if err != nil {
			return Files{}, fmt.Errorf("download %s: %w", part.name, err)
		}

		*part.data = b
	}

	return f, nil
}

// store caches the files; failures only mean no offline copy next time.
func store(cache Cache, f Files) {
	for _, part := range f.parts() {
		_ = cache.Put(part.name, *part.data)
	}
}

func fromCache(cache Cache) (Files, bool) {
	var f Files

	for _, part := range f.parts() {
		b, ok, err := cache.Get(part.name)
		if err != nil || !ok {
			return Files{}, false
		}

		*part.data = b
	}

	return f, true
}
