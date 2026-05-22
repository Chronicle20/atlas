package pngenc

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func makeFixture() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 17), G: uint8(y * 17), B: 128, A: 255})
		}
	}
	return img
}

func TestEncodeIsByteIdenticalAcrossRuns(t *testing.T) {
	img := makeFixture()
	var a, b bytes.Buffer
	if err := Encode(&a, img); err != nil {
		t.Fatalf("encode a: %v", err)
	}
	if err := Encode(&b, img); err != nil {
		t.Fatalf("encode b: %v", err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatalf("byte-identity broken: a=%d bytes b=%d bytes", a.Len(), b.Len())
	}
}
