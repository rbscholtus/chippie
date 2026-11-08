package crt

import (
	"testing"

	"github.com/rbscholtus/chippie/chip8"
)

func TestCopyFramebuffer(t *testing.T) {
	t.Parallel()

	m := chip8.New(chip8.DefaultProfile(chip8.ModernChip8))
	// Draw the font glyph "0" (0xF0 0x90 ...) at (0, 0).
	if err := m.LoadROM([]byte{0x60, 0x00, 0xF0, 0x29, 0xD0, 0x05, 0x12, 0x06}); err != nil {
		t.Fatal(err)
	}

	m.RunFrame()

	v := m.Display()
	pix := make([]byte, v.Width()*v.Height()*bytesPerPixel)

	if !CopyFramebuffer(v, &DefaultPalette, pix) {
		t.Fatal("CopyFramebuffer refused a big enough buffer")
	}

	at := func(x, y int) byte { return pix[(y*v.Width()+x)*bytesPerPixel] }

	if at(0, 0) != 0xFF || at(3, 0) != 0xFF || at(4, 0) != 0x00 || at(1, 1) != 0x00 {
		t.Errorf("unexpected pixels: (0,0)=%#x (3,0)=%#x (4,0)=%#x (1,1)=%#x", at(0, 0), at(3, 0), at(4, 0), at(1, 1))
	}

	if pix[3] != 0xFF {
		t.Error("alpha should be opaque")
	}

	if CopyFramebuffer(v, &DefaultPalette, pix[:10]) {
		t.Error("CopyFramebuffer accepted a buffer that's too small")
	}
}

func TestFillStatic(t *testing.T) {
	t.Parallel()

	a, b := make([]byte, 64*32*bytesPerPixel), make([]byte, 64*32*bytesPerPixel)
	FillStatic(a, 1)
	FillStatic(b, 2)

	if string(a) == string(b) {
		t.Error("different seeds gave the same static")
	}

	grey := map[byte]bool{}

	for i := 0; i+bytesPerPixel <= len(a); i += bytesPerPixel {
		px := a[i : i+bytesPerPixel]
		if px[0] != px[1] || px[0] != px[2] || px[3] != 0xFF {
			t.Fatalf("pixel %d isn't opaque grey: %v", i/bytesPerPixel, px)
		}

		grey[px[0]] = true
	}

	if len(grey) < 100 {
		t.Errorf("only %d shades of grey; static should look random", len(grey))
	}
}

func BenchmarkCopyFramebuffer(b *testing.B) {
	v := chip8.New(chip8.DefaultProfile(chip8.ModernChip8)).Display()
	pix := make([]byte, v.Width()*v.Height()*bytesPerPixel)

	b.ReportAllocs()

	for b.Loop() {
		CopyFramebuffer(v, &DefaultPalette, pix)
	}
}
