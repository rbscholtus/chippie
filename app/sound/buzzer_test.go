package sound

import (
	"encoding/binary"
	"math"
	"testing"
)

// samples reads n stereo frames and returns the left channel.
func samples(b *Buzzer, n int) []float32 {
	buf := make([]byte, n*bytesPerFrame)

	got, _ := b.Read(buf)
	out := make([]float32, got/bytesPerFrame)

	for i := range out {
		left := math.Float32frombits(binary.LittleEndian.Uint32(buf[i*bytesPerFrame:]))
		right := math.Float32frombits(binary.LittleEndian.Uint32(buf[i*bytesPerFrame+4:]))

		if left != right {
			panic("channels differ")
		}

		out[i] = left
	}

	return out
}

func TestBuzzerSilentWhenOff(t *testing.T) {
	t.Parallel()

	var b Buzzer
	for i, s := range samples(&b, 1000) {
		if s != 0 {
			t.Fatalf("sample %d = %v, want silence", i, s)
		}
	}
}

func TestBuzzerSquareWave(t *testing.T) {
	t.Parallel()

	var b Buzzer
	b.Set(true)

	s := samples(&b, SampleRate/10) // 100 ms
	steady := s[rampSamples:]       // after the fade-in

	for i, v := range steady {
		if v != float32(Amplitude) && v != -float32(Amplitude) {
			t.Fatalf("sample %d = %v, want ±%v", i, v, Amplitude)
		}
	}

	// Count sign changes: two per period.
	changes := 0

	for i := 1; i < len(steady); i++ {
		if (steady[i] > 0) != (steady[i-1] > 0) {
			changes++
		}
	}

	periods := float64(changes) / 2
	want := float64(len(steady)) / float64(SampleRate) * Frequency

	if math.Abs(periods-want) > 1 {
		t.Errorf("%.1f periods in %d samples, want about %.1f", periods, len(steady), want)
	}
}

func TestBuzzerRamps(t *testing.T) {
	t.Parallel()

	var b Buzzer
	b.Set(true)

	s := samples(&b, rampSamples)
	if math.Abs(float64(s[0])) >= Amplitude/2 {
		t.Errorf("first sample %v: no fade-in", s[0])
	}

	b.Set(false)

	s = samples(&b, 2*rampSamples)
	if s[len(s)-1] != 0 {
		t.Errorf("last sample %v, want silence after fade-out", s[len(s)-1])
	}
}

func TestBuzzerWholeFrames(t *testing.T) {
	t.Parallel()

	var b Buzzer

	n, err := b.Read(make([]byte, 13))
	if n != bytesPerFrame || err != nil {
		t.Errorf("Read(13 bytes) = %d, %v; want %d, nil", n, err, bytesPerFrame)
	}
}

func BenchmarkBuzzerRead(b *testing.B) {
	var bz Buzzer
	bz.Set(true)

	buf := make([]byte, 4096)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = bz.Read(buf)
	}
}

func TestPatternRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pitch byte
		want  float64
	}{
		{64, 4000},
		{112, 8000},
		{16, 2000},
		{255, 4000 * 15.8},
	}

	for _, tt := range tests {
		if got := patternRate(tt.pitch); got < tt.want*0.99 || got > tt.want*1.01 {
			t.Errorf("pitch %d: %.1f, want about %.1f", tt.pitch, got, tt.want)
		}
	}
}

func TestPatternPlaysBitsTopFirst(t *testing.T) {
	t.Parallel()

	var (
		b       Buzzer
		pattern [PatternBytes]byte
	)

	// At the base pitch, each pattern bit lasts this many output samples.
	const samplesPerBit = SampleRate / basePatternRate

	pattern[0] = 0b1010_0000

	b.SetPattern(true, pattern, basePitch)
	b.Set(true)
	samples(&b, rampSamples*2) // past the fade-in

	b.patternPos = 0

	got := samples(&b, samplesPerBit*4)
	for i, wantHigh := range []bool{true, false, true, false} {
		if s := got[i*samplesPerBit+1]; (s > 0) != wantHigh {
			t.Errorf("bit %d: sample %v, want high=%v", i, s, wantHigh)
		}
	}
}

func TestPatternKeepsPositionAcrossUpdates(t *testing.T) {
	t.Parallel()

	var (
		b       Buzzer
		pattern [PatternBytes]byte
	)

	b.SetPattern(true, pattern, basePitch)
	b.Set(true)
	samples(&b, 100)

	before := b.patternPos
	if before == 0 {
		t.Fatal("pattern didn't advance")
	}

	b.SetPattern(true, pattern, basePitch)

	if b.patternPos != before {
		t.Errorf("position reset by an update: %v, was %v", b.patternPos, before)
	}
}

func TestPatternWrapsAndStaysInRange(t *testing.T) {
	t.Parallel()

	var (
		b       Buzzer
		pattern [PatternBytes]byte
	)

	b.SetPattern(true, pattern, 255)
	b.Set(true)
	samples(&b, SampleRate/10)

	if b.patternPos < 0 || b.patternPos >= patternBits {
		t.Errorf("position %v outside the pattern", b.patternPos)
	}
}

func TestWithoutPatternPlaysTheSquareWave(t *testing.T) {
	t.Parallel()

	var plain, cleared Buzzer

	plain.Set(true)
	cleared.SetPattern(false, [PatternBytes]byte{0xFF}, 200)
	cleared.Set(true)

	a, c := samples(&plain, 500), samples(&cleared, 500)
	for i := range a {
		if a[i] != c[i] {
			t.Fatalf("sample %d: %v, want the square wave's %v", i, c[i], a[i])
		}
	}
}

func TestBuzzerVolume(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ volume, want float64 }{
		{1, Amplitude}, {0.5, Amplitude * 0.25}, {0, 0}, {-1, 0}, {2, Amplitude},
	} {
		var b Buzzer
		b.SetVolume(tc.volume)
		b.Set(true)

		s := samples(&b, 1000)
		if got := math.Abs(float64(s[len(s)-1])); math.Abs(got-tc.want) > 1e-6 {
			t.Errorf("volume %v: peak %v, want %v", tc.volume, got, tc.want)
		}
	}
}
