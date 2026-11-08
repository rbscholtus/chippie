package chip8

import (
	"os"
	"strconv"
	"testing"
)

// coraxROM is a busy test ROM for benchmarks.
const coraxROM = "../third_party/chip8-test-suite/bin/3-corax+.ch8"

func loadBenchROM(b *testing.B) []byte {
	b.Helper()

	rom, err := os.ReadFile(coraxROM)
	if err != nil {
		b.Skipf("test suite not checked out: %v", err)
	}

	return rom
}

// BenchmarkRunFrame measures one frame per platform at the default tick
// rate and at 1000 ticks. Allocations are reported and tracked, not
// enforced.
func BenchmarkRunFrame(b *testing.B) {
	rom := loadBenchROM(b)

	for _, p := range []Platform{ModernChip8, OriginalChip8} {
		for _, ticks := range []int{p.DefaultTicks(), 1000} {
			b.Run(p.String()+"/"+strconv.Itoa(ticks), func(b *testing.B) {
				prof := DefaultProfile(p)
				prof.TicksPerFrame = ticks
				m := New(prof)

				if err := m.LoadROM(rom); err != nil {
					b.Fatal(err)
				}

				b.ReportAllocs()

				for b.Loop() {
					m.RunFrame()

					if m.Halted() {
						b.Fatal(m.State().Err)
					}
				}
			})
		}
	}
}

// BenchmarkDXYN measures drawing a 15-row lo-res sprite.
func BenchmarkDXYN(b *testing.B) {
	b.Run("lores", func(b *testing.B) {
		m := New(DefaultProfile(ModernChip8))
		m.i = FontAddr
		m.v[1], m.v[2] = 61, 20 // crosses the right edge

		b.ReportAllocs()

		for b.Loop() {
			m.pc = ProgramStart
			exec(m, 0xD12F)
		}
	})
}

// BenchmarkTrace measures a frame with tracing on.
func BenchmarkTrace(b *testing.B) {
	rom := loadBenchROM(b)
	prof := DefaultProfile(ModernChip8)
	prof.TicksPerFrame = 1000
	m := New(prof)
	m.EnableTrace(4096)

	if err := m.LoadROM(rom); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		m.RunFrame()
	}
}
