//go:build !(js && wasm)

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := &Cache{store: StoreROMs, dir: dir}

	if _, ok, err := c.Get("abc123"); ok || err != nil {
		t.Fatalf("empty cache: ok=%v err=%v", ok, err)
	}

	if err := c.Put("abc123", []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}

	got, ok, err := c.Get("abc123")
	if !ok || err != nil || string(got) != "\x01\x02\x03" {
		t.Errorf("Get = %v, %v, %v", got, ok, err)
	}

	// Stores are separate.
	if _, ok, _ := (&Cache{store: StoreMetadata, dir: dir}).Get("abc123"); ok {
		t.Error("value leaked into another store")
	}

	// Overwrite.
	if err := c.Put("abc123", []byte("new")); err != nil {
		t.Fatal(err)
	}

	if got, _, _ := c.Get("abc123"); string(got) != "new" {
		t.Errorf("after overwrite: %q", got)
	}
}

func TestCacheRejectsBadKeys(t *testing.T) {
	t.Parallel()

	c := &Cache{store: StoreROMs, dir: t.TempDir()}

	for _, key := range []string{"", "../escape", "a/b", "with space"} {
		if err := c.Put(key, nil); !errors.Is(err, ErrBadKey) {
			t.Errorf("Put(%q): err=%v, want ErrBadKey", key, err)
		}

		if _, _, err := c.Get(key); !errors.Is(err, ErrBadKey) {
			t.Errorf("Get(%q): err=%v, want ErrBadKey", key, err)
		}
	}
}

func TestPrefsPersist(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config", "prefs.json")
	p := newPrefs(path)
	p.Set("overrides.abc", `{"tickrate":30}`)
	p.Set("last", "game/danm8ku")
	p.Set("last", "") // deletes

	if !p.Persistent() {
		t.Fatal("prefs should be persistent with a writable config directory")
	}

	again := newPrefs(path)

	if v, ok := again.Get("overrides.abc"); !ok || v != `{"tickrate":30}` {
		t.Errorf("after reload: %q %v", v, ok)
	}

	if _, ok := again.Get("last"); ok {
		t.Error("deleted key came back")
	}
}

func TestPrefsFallBackToMemory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Make the config directory a file, so writing fails.
	if err := os.WriteFile(filepath.Join(dir, "config"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	p := newPrefs(filepath.Join(dir, "config", "prefs.json"))
	p.Set("k", "v")

	if v, ok := p.Get("k"); !ok || v != "v" {
		t.Errorf("value lost: %q %v", v, ok)
	}

	if p.Persistent() {
		t.Error("prefs claim to be persistent although writing failed")
	}

	if q := newPrefs(""); q.Persistent() {
		t.Error("prefs without a path claim to be persistent")
	}
}

func TestCacheUnavailableWithoutDirectory(t *testing.T) {
	t.Parallel()

	c := &Cache{store: StoreROMs}

	if err := c.Put("abc", nil); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Put: err=%v, want ErrUnavailable", err)
	}

	if _, _, err := c.Get("abc"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Get: err=%v, want ErrUnavailable", err)
	}
}

func TestPickFileUnsupportedNatively(t *testing.T) {
	t.Parallel()

	if CanPickFiles() {
		t.Error("native builds have no file picker")
	}

	if r := <-PickFile(); !errors.Is(r.Err, ErrUnsupported) {
		t.Errorf("err=%v, want ErrUnsupported", r.Err)
	}
}
