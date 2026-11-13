package app

import (
	"testing"

	"github.com/rbscholtus/chippie/chip8"
)

func newTestSettings() *settings {
	own := chip8.DefaultProfile(chip8.ModernChip8)
	own.TicksPerFrame = 200

	return newSettings(own, own, chip8.DefaultProfile, false)
}

func rowOf(s *settings, kind settingKind) int {
	for i, r := range s.rows {
		if r.kind == kind {
			return i
		}
	}

	return -1
}

func TestSettingsSpeed(t *testing.T) {
	t.Parallel()

	s := newTestSettings()
	speed := rowOf(s, settingSpeed)

	if eff := s.change(speed, 1); eff != effectLive || s.profile.TicksPerFrame != 300 {
		t.Errorf("faster: %v, %d", eff, s.profile.TicksPerFrame)
	}

	s.change(speed, -1)
	s.change(speed, -1)

	if s.profile.TicksPerFrame != 100 {
		t.Errorf("slower twice: %d, want 100", s.profile.TicksPerFrame)
	}

	if l := s.label(speed); l.value != "100 per frame" || !l.changed {
		t.Errorf("label %+v", l)
	}

	for range 30 {
		s.change(speed, -1)
	}

	if s.profile.TicksPerFrame != 1 {
		t.Errorf("slowest: %d", s.profile.TicksPerFrame)
	}

	if stepSpeed(10000, 1) != 10000 || stepSpeed(13, 1) != 15 || stepSpeed(13, -1) != 12 {
		t.Error("stepSpeed edges wrong")
	}
}

func TestSettingsQuirkToggles(t *testing.T) {
	t.Parallel()

	s := newTestSettings()
	first := rowOf(s, settingQuirk) // VF reset

	if eff := s.change(first, 0); eff != effectLive || !s.profile.Quirks.Logic {
		t.Errorf("toggle on: %v %+v", eff, s.profile.Quirks)
	}

	if l := s.label(first); l.name != "VF reset after AND/OR/XOR" || l.value != "on" || !l.changed {
		t.Errorf("label %+v", l)
	}

	s.change(first, 0)

	if s.profile.Quirks.Logic {
		t.Error("toggle off failed")
	}

	// Every quirk row toggles its own field.
	for i, row := range s.rows {
		if row.kind != settingQuirk {
			continue
		}

		before := s.profile.Quirks
		s.change(i, 0)

		if s.profile.Quirks == before {
			t.Errorf("row %d (%s) changed nothing", i, quirkSettings[row.quirk].id)
		}
	}
}

func TestSettingsPlatformAndReset(t *testing.T) {
	t.Parallel()

	s := newTestSettings()
	plat := rowOf(s, settingPlatform)

	if eff := s.change(plat, -1); eff != effectRestart || s.profile.Platform != chip8.OriginalChip8 {
		t.Fatalf("next platform: %v %s", eff, s.profile.Platform)
	}

	// The new platform's recommended quirks come with it.
	if !s.profile.Quirks.VBlank || !s.profile.Quirks.Logic {
		t.Errorf("VIP quirks not applied: %+v", s.profile.Quirks)
	}

	if l := s.label(plat); !l.changed {
		t.Error("platform should show as changed")
	}

	if eff := s.change(rowOf(s, settingReset), 0); eff != effectRestart || s.profile != s.own {
		t.Errorf("reset: %v %+v", eff, s.profile)
	}

	s.change(rowOf(s, settingSpeed), 1)

	if eff := s.change(rowOf(s, settingReset), 0); eff != effectLive {
		t.Errorf("reset on the same platform should be live: %v", eff)
	}

	last := stepPlatform(chip8.OriginalChip8, -1)

	wraps := last != chip8.OriginalChip8 && last.Implemented() && stepPlatform(last, 1) == chip8.OriginalChip8
	if !wraps {
		t.Error("stepPlatform should wrap between the implemented platforms")
	}
}

func TestSettingsTraceRows(t *testing.T) {
	t.Parallel()

	s := newTestSettings()
	if rowOf(s, settingCopyTrace) >= 0 {
		t.Error("copy row shown without tracing")
	}

	if eff := s.change(rowOf(s, settingTrace), 0); eff != effectTrace || !s.trace {
		t.Fatalf("trace on: %v", eff)
	}

	copyRow := rowOf(s, settingCopyTrace)
	if copyRow < 0 || s.change(copyRow, 0) != effectCopy {
		t.Error("copy row missing or wrong")
	}

	if s.change(rowOf(s, settingBack), 0) != effectClose || s.change(99, 0) != effectNone {
		t.Error("back or out-of-range row wrong")
	}

	s.selected = len(s.rows) - 1
	s.move(1)

	if s.selected != 0 || !s.steps(rowOf(s, settingSpeed)) || s.steps(rowOf(s, settingBack)) {
		t.Error("move or steps wrong")
	}
}

func TestSettingsLegacySuperChip(t *testing.T) {
	t.Parallel()

	legacyRow := func(s *settings) int {
		for i, r := range s.rows {
			if r.kind == settingQuirk && quirkSettings[r.quirk].id == legacyQuirkID {
				return i
			}
		}

		return -1
	}

	prof := chip8.DefaultProfile(chip8.ModernChip8)
	schip := chip8.DefaultProfile(chip8.SuperChip)
	forPlatform := chip8.DefaultProfile

	s := newSettings(prof, prof, forPlatform, false)
	if legacyRow(s) != -1 {
		t.Error("legacy row shown on modern CHIP-8")
	}

	s = newSettings(schip, schip, forPlatform, false)

	i := legacyRow(s)
	if i < 0 {
		t.Fatal("no legacy row on SUPER-CHIP")
	}

	if eff := s.change(i, 0); eff != effectRestart || !s.profile.Quirks.LegacySuperChip {
		t.Errorf("toggle: effect %v, quirk %v", eff, s.profile.Quirks.LegacySuperChip)
	}

	if !s.label(i).changed {
		t.Error("legacy on should show as changed")
	}

	if eff := s.change(rowOf(s, settingReset), 0); eff != effectRestart || s.profile.Quirks.LegacySuperChip {
		t.Errorf("reset: effect %v, quirk %v", eff, s.profile.Quirks.LegacySuperChip)
	}

	if s.change(rowOf(s, settingPlatform), 1); s.profile.Platform.HasLegacyMode() && legacyRow(s) == -1 {
		t.Error("legacy row missing after stepping the platform")
	}
}
