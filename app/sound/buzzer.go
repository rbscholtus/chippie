// Package sound plays the CHIP-8 buzzer through Ebitengine's audio package.
package sound

import (
	"encoding/binary"
	"math"
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
)

// Buzzer is an endless audio stream: a square wave while it is on, silence
// while it is off. It is an [io.Reader] of little-endian float32 stereo
// samples, as Ebitengine's audio.NewPlayerF32 expects. Set is safe to call
// from the game loop while the audio goroutine reads.
type Buzzer struct {
	on    atomic.Bool
	phase int     // sample position within one wave period
	level float32 // current envelope level, 0..1
}

// Set switches the tone on or off.
func (b *Buzzer) Set(on bool) { b.on.Store(on) }

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

	n := len(p) / bytesPerFrame * bytesPerFrame

	for i := 0; i < n; i += bytesPerFrame {
		switch {
		case b.level < target:
			b.level = min(target, b.level+step)
		case b.level > target:
			b.level = max(target, b.level-step)
		}

		sample := float32(Amplitude) * b.level
		if b.phase >= period/2 {
			sample = -sample
		}

		b.phase = (b.phase + 1) % period

		bits := math.Float32bits(sample)
		binary.LittleEndian.PutUint32(p[i:], bits)
		binary.LittleEndian.PutUint32(p[i+4:], bits)
	}

	return n, nil
}
