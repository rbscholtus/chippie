package app

import (
	"testing"

	"github.com/rbscholtus/chippie/internal/chip8"
)

// fakeFlags is a machine whose flags the test sets, counting saves.
type fakeFlags struct {
	flags [chip8.NumFlags]byte
	saves uint32
}

func (f *fakeFlags) Flags() [chip8.NumFlags]byte { return f.flags }
func (f *fakeFlags) FlagSaves() uint32           { return f.saves }

// save is a program running FX75.
func (f *fakeFlags) save(b ...byte) {
	f.flags = [chip8.NumFlags]byte{}
	copy(f.flags[:], b)
	f.saves++
}

// fakePrefs records preference writes.
type fakePrefs struct {
	values map[string]string
	writes int
}

func newFakePrefs() *fakePrefs { return &fakePrefs{values: map[string]string{}} }

func (p *fakePrefs) Get(k string) (string, bool) { v, ok := p.values[k]; return v, ok }

func (p *fakePrefs) Set(k, v string) {
	p.writes++
	if v == "" {
		delete(p.values, k)
	} else {
		p.values[k] = v
	}
}

const testSHA1 = "268de0f4c377e4e41a5bca78f5f82cf826d91a64"

func TestFlagSaverWritesASave(t *testing.T) {
	t.Parallel()

	m, prefs := &fakeFlags{}, newFakePrefs()

	var s flagSaver
	s.start(testSHA1, [chip8.NumFlags]byte{})

	s.frame(m, prefs)

	if prefs.writes != 0 {
		t.Fatalf("%d writes without a save", prefs.writes)
	}

	m.save(0xD3, 0x90, 0x01, 0x30)
	s.frame(m, prefs)

	if got, want := prefs.values["flags/"+testSHA1], "d3900130000000000000000000000000"; got != want {
		t.Errorf("saved %q, want %q", got, want)
	}

	if got := loadFlags(prefs, testSHA1); got != m.flags {
		t.Errorf("loadFlags = % X, want % X", got, m.flags)
	}
}

func TestFlagSaverWritesAtMostOnceASecond(t *testing.T) {
	t.Parallel()

	m, prefs := &fakeFlags{}, newFakePrefs()

	var s flagSaver
	s.start(testSHA1, [chip8.NumFlags]byte{})

	// A program saving different bytes every frame for 2 s.
	for i := range 120 {
		m.save(byte(i + 1))
		s.frame(m, prefs)
	}

	if prefs.writes > 2 {
		t.Errorf("%d writes in 120 frames, want at most 2", prefs.writes)
	}

	// The last save is written once the wait is over, or on flush.
	s.flush(prefs)

	if got := loadFlags(prefs, testSHA1); got[0] != 120 {
		t.Errorf("after flush, flag 0 = %d, want the last save (120)", got[0])
	}
}

func TestFlagSaverSkipsUnchangedSaves(t *testing.T) {
	t.Parallel()

	m, prefs := &fakeFlags{}, newFakePrefs()
	restored := [chip8.NumFlags]byte{7}

	var s flagSaver
	s.start(testSHA1, restored)

	for range 200 {
		m.save(7) // the same bytes as restored
		s.frame(m, prefs)
	}

	s.flush(prefs)

	if prefs.writes != 0 {
		t.Errorf("%d writes of unchanged flags, want none", prefs.writes)
	}
}

func TestFlagSaverZeroDeletes(t *testing.T) {
	t.Parallel()

	m, prefs := &fakeFlags{}, newFakePrefs()
	prefs.values["flags/"+testSHA1] = "01000000000000000000000000000000"

	var s flagSaver
	s.start(testSHA1, loadFlags(prefs, testSHA1))

	m.save() // all zero
	s.frame(m, prefs)

	if _, ok := prefs.values["flags/"+testSHA1]; ok {
		t.Error("all-zero flags still stored")
	}
}

func TestFlagSaverIdleWithoutROM(t *testing.T) {
	t.Parallel()

	m, prefs := &fakeFlags{}, newFakePrefs()

	var s flagSaver

	m.save(1)
	s.frame(m, prefs)
	s.flush(prefs)

	if prefs.writes != 0 {
		t.Errorf("%d writes with no ROM started", prefs.writes)
	}
}

func TestLoadFlagsIgnoresBadValues(t *testing.T) {
	t.Parallel()

	prefs := newFakePrefs()

	for _, bad := range []string{"zz", "0102", "d39001300000000000000000000000000000ff"} {
		prefs.values["flags/"+testSHA1] = bad
		if got := loadFlags(prefs, testSHA1); got != ([chip8.NumFlags]byte{}) {
			t.Errorf("loadFlags(%q) = % X, want zeros", bad, got)
		}
	}
}
