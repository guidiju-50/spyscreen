package encoder

import (
	"bytes"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestJPEGEncode(t *testing.T) {
	enc := NewJPEG(80)
	frame := FillSolidRGBA(32, 32, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	out, err := enc.Encode(frame)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 100 {
		t.Fatalf("jpeg too small: %d", len(out))
	}
	_, err = jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
}
