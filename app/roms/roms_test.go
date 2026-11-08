package roms

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rbscholtus/chippie/chip8"
	"github.com/rbscholtus/chippie/platform"
)

const suiteBin = "../../third_party/chip8-test-suite/bin"

func newTestLibrary(t *testing.T) *Library {
	t.Helper()

	l, err := NewLibrary(suiteBin)
	if err != nil {
		t.Fatal(err)
	}

	return l
}

func TestBundledGames(t *testing.T) {
	t.Parallel()

	l := newTestLibrary(t)
	playable := map[string]bool{}

	for _, e := range l.Entries() {
		if e.Group != Games {
			continue
		}

		if _, err := l.Load(t.Context(), e); err != nil {
			t.Errorf("%s: %v", e.ID, err)
		}

		if e.Title == "" || len(e.Authors) == 0 || e.Ticks <= 0 {
			t.Errorf("%s: incomplete entry %+v", e.ID, e)
		}

		playable[e.ID] = e.Playable()
	}

	want := map[string]bool{
		"game/glitchGhost": true, "game/danm8ku": true,
		"game/turnover77": false, "game/octopeg": false,
		"game/redOctober": false, "game/anEveningToDieFor": false,
	}

	for id, p := range want {
		if got, ok := playable[id]; !ok || got != p {
			t.Errorf("%s: playable=%v (present %v), want %v", id, got, ok, p)
		}
	}

	if len(playable) != len(want) {
		t.Errorf("%d bundled games, want %d", len(playable), len(want))
	}
}

// TestPlayableGamesRun runs each playable game for ten seconds of frames,
// with a few keys pressed, and checks it doesn't halt.
func TestPlayableGamesRun(t *testing.T) {
	t.Parallel()

	l := newTestLibrary(t)

	for _, e := range l.Entries() {
		if e.Group != Games || !e.Playable() {
			continue
		}

		t.Run(e.ID, func(t *testing.T) {
			t.Parallel()

			rom, err := l.Load(t.Context(), e)
			if err != nil {
				t.Fatal(err)
			}

			m := chip8.New(e.Profile())
			if err := m.LoadROM(rom); err != nil {
				t.Fatal(err)
			}

			for f := range 600 {
				key := uint8(f / 60 % chip8.NumKeys)
				m.SetKey(key, f%60 < 30)
				m.RunFrame()

				if err := m.State().Err; err != nil {
					t.Fatalf("frame %d: %v", f, err)
				}
			}
		})
	}
}

func TestTestSuiteEntries(t *testing.T) {
	t.Parallel()

	l := newTestLibrary(t)

	e, ok := l.Find("test/2-ibm-logo")
	if !ok || !e.Playable() || e.Group != TestSuite {
		t.Fatalf("test/2-ibm-logo: %+v %v", e, ok)
	}

	if _, err := l.Load(t.Context(), e); err != nil {
		t.Error(err)
	}

	if s, _ := l.Find("test/8-scrolling"); s.Playable() {
		t.Error("the scrolling test needs SUPER-CHIP and shouldn't be playable yet")
	}

	if _, ok := l.Find("nope"); ok {
		t.Error("Find returned an entry for an unknown ID")
	}
}

func TestOfflineAndAliases(t *testing.T) {
	t.Parallel()

	l := newTestLibrary(t)
	offline := l.Offline()

	if len(offline) != 14 || !offline["b9bbc12cee3f7b9d3b1f69161f7d7a2d86953379"] {
		t.Errorf("offline: %d hashes", len(offline))
	}

	if h, ok := l.HashFor("game/glitchGhost"); !ok || h != "2cdcb3c29a5f013a991db5909ca8e18e27b3c42b" {
		t.Errorf("HashFor(game/glitchGhost) = %q %v", h, ok)
	}

	if rom, ok := l.LoadHash(t.Context(), "B9BBC12CEE3F7B9D3B1F69161F7D7A2D86953379"); !ok || len(rom) == 0 {
		t.Error("LoadHash of the IBM test failed")
	}

	if _, ok := l.LoadHash(t.Context(), "0000"); ok {
		t.Error("LoadHash of an unknown hash succeeded")
	}
}

func TestLoadChecksHash(t *testing.T) {
	t.Parallel()

	l := newTestLibrary(t)
	e, _ := l.Find("test/2-ibm-logo")
	e.SHA1 = strings.Repeat("0", 40)

	if _, err := l.Load(t.Context(), e); !errors.Is(err, ErrHashMismatch) {
		t.Errorf("err=%v, want ErrHashMismatch", err)
	}
}

func TestLoadFromURL(t *testing.T) {
	t.Parallel()

	rom, err := os.ReadFile(filepath.Join(suiteBin, "2-ibm-logo.ch8"))
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/roms/test-suite/2-ibm-logo.ch8":
			_, _ = w.Write(rom)
		case "/roms/test-suite/big.ch8":
			_, _ = w.Write(make([]byte, maxROMSize+1))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	l, err := NewLibrary(srv.URL + "/roms/test-suite/")
	if err != nil {
		t.Fatal(err)
	}

	e, _ := l.Find("test/2-ibm-logo")

	res := <-l.LoadAsync(t.Context(), e)
	if res.Err != nil || len(res.ROM) != len(rom) || res.Entry.ID != e.ID {
		t.Errorf("LoadAsync: %d bytes, err=%v", len(res.ROM), res.Err)
	}

	missing, _ := l.Find("test/1-chip8-logo")
	if _, err := l.Load(t.Context(), missing); !errors.Is(err, platform.ErrHTTPStatus) {
		t.Errorf("missing file: err=%v, want ErrHTTPStatus", err)
	}

	big := e
	big.File, big.SHA1 = "big.ch8", ""

	if _, err := l.Load(t.Context(), big); !errors.Is(err, platform.ErrTooLarge) {
		t.Errorf("oversized: err=%v, want ErrTooLarge", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := l.Load(ctx, e); err == nil {
		t.Error("cancelled load succeeded")
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()

	e, rom, err := ReadFile(filepath.Join(suiteBin, "2-ibm-logo.ch8"))
	if err != nil || len(rom) == 0 || e.Title != "2-ibm-logo.ch8" || e.Platform != chip8.ModernChip8 {
		t.Errorf("ReadFile: %+v %d %v", e, len(rom), err)
	}

	big := filepath.Join(t.TempDir(), "big.ch8")
	if err := os.WriteFile(big, make([]byte, maxROMSize+1), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := ReadFile(big); !errors.Is(err, ErrTooLarge) {
		t.Errorf("oversized: err=%v", err)
	}

	if _, _, err := ReadFile("does-not-exist"); err == nil {
		t.Error("missing file: no error")
	}
}
