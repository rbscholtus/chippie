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
