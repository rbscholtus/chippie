package platform

import (
	"errors"
	"regexp"
)

// Cache stores, by key, the bytes the app downloads, so they work offline
// after the first time: ROMs (keyed by SHA-1) and catalogue metadata. In
// the browser it is IndexedDB; natively, files in the user's cache
// directory. Get and Put may block on I/O: call them from a goroutine,
// never from Update or Draw.
type Cache struct {
	store string
	dir   string // native only: the cache directory
}

// Cache stores.
const (
	StoreROMs     = "roms"
	StoreMetadata = "metadata"
)

// Errors from the cache.
var (
	ErrBadKey      = errors.New("platform: bad cache key")
	ErrUnavailable = errors.New("platform: storage unavailable")
)

// validKey keeps keys safe as file names and IndexedDB keys.
var validKey = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

// NewCache returns the cache for one store (StoreROMs or StoreMetadata).
func NewCache(store string) *Cache { return &Cache{store: store, dir: defaultCacheDir()} }

// Get returns the value for key; the bool is false if there is none.
func (c *Cache) Get(key string) ([]byte, bool, error) {
	if !validKey.MatchString(key) {
		return nil, false, ErrBadKey
	}

	return c.get(key)
}

// Put stores value under key.
func (c *Cache) Put(key string, value []byte) error {
	if !validKey.MatchString(key) {
		return ErrBadKey
	}

	return c.put(key, value)
}
