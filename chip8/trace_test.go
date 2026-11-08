package chip8

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite generated golden files")

func TestTraceRingBuffer(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	if m.Trace() != nil {
		t.Fatal("trace should be off by default")
	}

	m.EnableTrace(4)
	mustLoad(t, m, program(0x6001, 0x6102, 0x6203, 0x6304, 0x6405, 0x1200))

	m.RunFrame() // 12 instructions through a 6-instruction loop

	tr := m.Trace()
	if tr.Len() != 4 {
		t.Fatalf("Len=%d, want 4", tr.Len())
	}

	got := tr.Entries()
	want := []uint16{0x6203, 0x6304, 0x6405, 0x1200} // the last four, oldest first

	for i, e := range got {
		if e.Op != want[i] || e.Frame != 1 {
			t.Errorf("entry %d: %+v, want op %04X in frame 1", i, e, want[i])
		}
	}

	if got[0].PC != ProgramStart+4 {
		t.Errorf("first entry PC=%03X, want 204", got[0].PC)
	}

	m.Reset(m.Profile())

	if m.Trace().Len() != 0 {
		t.Error("Reset should clear the trace")
	}

	m.EnableTrace(0)

	if m.Trace() != nil {
		t.Error("EnableTrace(0) should switch tracing off")
	}
}

func TestTraceBeforeWrap(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	m.EnableTrace(100)
	mustLoad(t, m, program(0x1200))
	m.RunFrame()

	if n := len(m.Trace().Entries()); n != 12 {
		t.Errorf("%d entries, want 12", n)
	}
}

// TestDisassemble compares the disassembly of every base opcode form with
// testdata/disasm-<platform>.txt. Run with -update to rewrite it.
func TestDisassemble(t *testing.T) {
	t.Parallel()

	ops := []uint16{
		0x00E0, 0x00EE, 0x0123, 0x1ABC, 0x2ABC, 0x31AB, 0x41AB, 0x5120, 0x5121,
		0x61AB, 0x71AB, 0x8120, 0x8121, 0x8122, 0x8123, 0x8124, 0x8125, 0x8126,
		0x8127, 0x812E, 0x8128, 0x9120, 0x9121, 0xAABC, 0xBABC, 0xC1AB, 0xD12F,
		0xE19E, 0xE1A1, 0xE1FF, 0xF107, 0xF10A, 0xF115, 0xF118, 0xF11E, 0xF129,
		0xF133, 0xF155, 0xF165, 0xF1FF,
	}

	for _, p := range []Platform{ModernChip8, OriginalChip8} {
		entries := make([]TraceEntry, len(ops))
		for i, op := range ops {
			entries[i] = TraceEntry{PC: ProgramStart + 2*uint16(i), Op: op, I: 0x300, Frame: 1}
		}

		got := FormatTrace(p, entries)
		path := filepath.Join("testdata", "disasm-"+p.String()+".txt")

		if *update {
			if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
				t.Fatal(err)
			}
		}

		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run go test ./chip8 -run TestDisassemble -update)", err)
		}

		if got != string(want) {
			t.Errorf("%s disassembly differs from %s:\n%s", p, path, got)
		}
	}
}
