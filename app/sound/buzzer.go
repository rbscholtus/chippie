// Package sound plays the CHIP-8 buzzer and XO-CHIP's sampled audio through
// Ebitengine's audio package.
package sound

import (
	"encoding/binary"
	"math"
	"sync"
	"sync/atomic"
)

// Buzzer tone and stream format.
const (
	SampleRate = 48000 // Hz
	Frequency  = 440   // Hz; a plain square wave, like the COSMAC VIP's
	Amplitude  = 0.15  // peak level, well below clipping
	// rampSamples is the fade in/out length (2 ms), which avoids clicks.
	rampSamples = SampleRate / 500
	// bytesPerFrame is one stereo frame of float32 samples.
	bytesPerFrame = 8
	// PatternBytes is the size of XO-CHIP's audio pattern.
	PatternBytes = 16

	patternBits = PatternBytes * 8
	// XO-CHIP's playback rate: basePatternRate at basePitch, doubling every
	// pitchOctave steps.
	basePatternRate = 4000 // bits per second
	basePitch       = 64
	pitchOctave     = 48
)

// patternRate is how many pattern bits play per second at a pitch.
func patternRate(pitch byte) float64 {
	return basePatternRate * math.Pow(2, (float64(pitch)-basePitch)/pitchOctave)
}

// Buzzer is an endless audio stream: while it is on, a square wave or
// XO-CHIP's pattern; silence while it is off. It is an [io.Reader] of little-endian float32 stereo
// samples, as Ebitengine's audio.NewPlayerF32 expects. Set is safe to call
// from the game loop while the audio goroutine reads.
type Buzzer struct {
	on    atomic.Bool
	quiet atomic.Uint32 // float32 bits of 1 - volume, so the zero value is full volume
	phase int           // sample position within one wave period
	level float32       // current envelope level, 0..1

	patternPos float64 // bit position within the pattern, kept across updates

	mu         sync.Mutex // guards the pattern, set by the game loop
	hasPattern bool
	pattern    [PatternBytes]byte
	step       float64 // pattern bits per output sample
}

// Set switches the tone on or off.
func (b *Buzzer) Set(on bool) { b.on.Store(on) }

// SetPattern selects what plays while the buzzer is on: with has, the 128-bit
// pattern (first sample in the top bit) at the pitch; otherwise the square
// wave.
func (b *Buzzer) SetPattern(has bool, pattern [PatternBytes]byte, pitch byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.hasPattern, b.pattern = has, pattern
	b.step = patternRate(pitch) / SampleRate
}

// SetVolume sets the loudness, 0 (silent) to 1 (full). The knob's position
// is squared, so equal turns sound like equal steps.
func (b *Buzzer) SetVolume(v float64) {
	v = min(max(v, 0), 1)
	b.quiet.Store(math.Float32bits(float32(1 - v*v)))
}

// On reports whether the tone is switched on.
func (b *Buzzer) On() bool { return b.on.Load() }

// Read fills p with whole stereo frames. It never fails and never ends.
func (b *Buzzer) Read(p []byte) (int, error) {
	const (
		period = SampleRate / Frequency
		step   = 1.0 / rampSamples
	)

	target := float32(0)
	if b.on.Load() {
		target = 1
	}

	b.mu.Lock()
	hasPattern, pattern, patternStep := b.hasPattern, b.pattern, b.step
	b.mu.Unlock()

	n := len(p) / bytesPerFrame * bytesPerFrame
	gain := float32(Amplitude) * (1 - math.Float32frombits(b.quiet.Load()))

	for i := 0; i < n; i += bytesPerFrame {
		switch {
		case b.level < target:
			b.level = min(target, b.level+step)
		case b.level > target:
			b.level = max(target, b.level-step)
		}

		sample := gain * b.level

		if hasPattern {
			bit := int(b.patternPos)
			if pattern[bit/8]&(0x80>>(bit%8)) == 0 {
				sample = -sample
			}

			b.patternPos = math.Mod(b.patternPos+patternStep, patternBits)
		} else {
			if b.phase >= period/2 {
				sample = -sample
			}

			b.phase = (b.phase + 1) % period
		}

		bits := math.Float32bits(sample)
		binary.LittleEndian.PutUint32(p[i:], bits)
		binary.LittleEndian.PutUint32(p[i+4:], bits)
	}

	return n, nil
}
