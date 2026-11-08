package sound

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// bufferDuration keeps buzzer latency low; the tone is gated per frame.
const bufferDuration = 50 * time.Millisecond

// Output owns the audio context and the buzzer's player. Browsers only
// allow audio after a user gesture, so Start is called on the first click or
// key press, not at start-up.
type Output struct {
	buzzer Buzzer
	player *audio.Player
}

// Start creates the audio context and starts the buzzer stream. Calling it
// again does nothing.
func (o *Output) Start() error {
	if o.player != nil {
		return nil
	}

	ctx := audio.NewContext(SampleRate)

	p, err := ctx.NewPlayerF32(&o.buzzer)
	if err != nil {
		return fmt.Errorf("start audio: %w", err)
	}

	p.SetBufferSize(bufferDuration)
	p.Play()
	o.player = p

	return nil
}

// Started reports whether audio is running.
func (o *Output) Started() bool { return o.player != nil }

// SetBuzzer switches the tone on or off. It works before Start too; the
// tone is simply not heard yet.
func (o *Output) SetBuzzer(on bool) { o.buzzer.Set(on) }
