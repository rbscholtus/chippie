//go:build !(js && wasm)

package platform

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// defaultCacheDir is Chippie's folder in the user's cache directory, or ""
// if there is none (then the cache is unavailable).
func defaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}

	return filepath.Join(dir, "chippie")
}

func (c *Cache) get(key string) ([]byte, bool, error) {
	if c.dir == "" {
		return nil, false, ErrUnavailable
	}

	data, err := os.ReadFile(filepath.Join(c.dir, c.store, key))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("cache get: %w", err)
	}

	return data, true, nil
}

func (c *Cache) put(key string, value []byte) error {
	if c.dir == "" {
		return ErrUnavailable
	}

	dir := filepath.Join(c.dir, c.store)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("cache put: %w", err)
	}

	// Write to a temporary file and rename, so a crash never leaves half a
	// value behind.
	tmp, err := os.CreateTemp(dir, key+".*")
	if err != nil {
		return fmt.Errorf("cache put: %w", err)
	}

	if _, err := tmp.Write(value); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())

		return fmt.Errorf("cache put: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())

		return fmt.Errorf("cache put: %w", err)
	}

	if err := os.Rename(tmp.Name(), filepath.Join(dir, key)); err != nil {
		return fmt.Errorf("cache put: %w", err)
	}

	return nil
}
