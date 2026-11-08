// Package romstore downloads ROMs, checks their SHA-1 and caches them, so
// a ROM works offline after its first play (README §9.3).
package romstore

import (
	"context"
	"crypto/sha1" //nolint:gosec // G505: SHA-1 identifies ROMs, as in chip-8-database; not for security
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/rbscholtus/chippie/catalogue"
	"github.com/rbscholtus/chippie/platform"
)

// MaxROMSize bounds a download: 64 KB is the largest memory of any
// supported platform.
const MaxROMSize = 64 << 10

// Errors from Get.
var (
	ErrHashMismatch = errors.New("romstore: download doesn't match its SHA-1")
	ErrUnavailable  = errors.New("romstore: ROM unavailable")
)

// Cache keeps downloaded ROMs by SHA-1; *platform.Cache implements it.
type Cache interface {
	Get(key string) ([]byte, bool, error)
	Put(key string, value []byte) error
}

// FetchFunc downloads a URL, at most limit bytes; platform.Fetch by default.
type FetchFunc func(ctx context.Context, url string, limit int) ([]byte, error)

// LocalFunc returns a ROM that needs no download (bundled, or shipped beside
// the app), or false.
type LocalFunc func(ctx context.Context, sha1 string) ([]byte, bool)

// Store finds ROMs: cache first, then each download location in order.
type Store struct {
	urls  func(sha1 string) []catalogue.Location
	cache Cache
	fetch FetchFunc
	local LocalFunc
	base  string // resolves relative URLs (ROMs Chippie hosts itself)
}

// New returns a store. urls gives the download locations for a hash, best
// first (catalogue.Sources.URLs); base resolves relative URLs.
func New(urls func(string) []catalogue.Location, cache Cache, base string) *Store {
	return &Store{urls: urls, cache: cache, fetch: platform.Fetch, base: base}
}

// WithLocal sets where ROMs that need no download come from. They are
// tried after the cache and before any download.
func (s *Store) WithLocal(f LocalFunc) *Store {
	s.local = f

	return s
}

// WithFetch replaces the downloader (tests).
func (s *Store) WithFetch(f FetchFunc) *Store {
	s.fetch = f

	return s
}

// Result is a ROM and where it came from ("cache" or a source id).
type Result struct {
	SHA1 string
	ROM  []byte
	From string
	Err  error
}

// Result.From for ROMs that weren't downloaded.
const (
	FromCache = "cache"
	FromLocal = "local"
)

// Get finds a ROM in a goroutine and delivers the result on the returned
// channel, which has room for it: the game loop polls it without waiting.
func (s *Store) Get(ctx context.Context, sha1 string) <-chan Result {
	ch := make(chan Result, 1)

	go func() { ch <- s.Find(ctx, sha1) }()

	return ch
}

// Find looks in the cache, then tries each location until a download
// matches the hash, and caches it. It blocks; use Get from the game loop.
func (s *Store) Find(ctx context.Context, sha1 string) Result {
	key := strings.ToLower(sha1)

	if rom, ok, err := s.cache.Get(key); err == nil && ok && hashOf(rom) == key {
		return Result{SHA1: key, ROM: rom, From: FromCache}
	}

	if s.local != nil {
		if rom, ok := s.local(ctx, key); ok && hashOf(rom) == key {
			return Result{SHA1: key, ROM: rom, From: FromLocal}
		}
	}

	locs := s.urls(key)
	if len(locs) == 0 {
		return Result{SHA1: key, Err: fmt.Errorf("%w: %s: no download location", ErrUnavailable, key)}
	}

	var errs []error

	for _, loc := range locs {
		if err := ctx.Err(); err != nil {
			return Result{SHA1: key, Err: fmt.Errorf("%w: %w", ErrUnavailable, err)}
		}

		rom, err := s.fetch(ctx, s.resolve(loc.URL), MaxROMSize)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", loc.Source, err))

			continue
		}

		if got := hashOf(rom); got != key {
			errs = append(errs, fmt.Errorf("%s: %w: got %s", loc.Source, ErrHashMismatch, got))

			continue
		}

		// A failing cache only costs a download next time.
		_ = s.cache.Put(key, rom)

		return Result{SHA1: key, ROM: rom, From: loc.Source}
	}

	return Result{SHA1: key, Err: fmt.Errorf("%w: %s: %w", ErrUnavailable, key, errors.Join(errs...))}
}

// resolve makes a relative URL absolute against the app's base URL.
func (s *Store) resolve(u string) string {
	if strings.Contains(u, "://") || s.base == "" {
		return u
	}

	return strings.TrimSuffix(s.base, "/") + "/" + strings.TrimPrefix(u, "/")
}

// hashOf returns the lower-case hex SHA-1 of b.
func hashOf(b []byte) string {
	sum := sha1.Sum(b) //nolint:gosec // G401: see import

	return hex.EncodeToString(sum[:])
}

// Hash returns the lower-case hex SHA-1 of a ROM: how ROMs are identified.
func Hash(rom []byte) string { return hashOf(rom) }
