package canvas

import (
	"bytes"
	"compress/zlib"
	"image/color"
	"testing"
)

func TestExpand4to8(t *testing.T) {
	tests := []struct {
		input byte
		want  byte
	}{
		{0x0, 0x00},
		{0x5, 0x55},
		{0xA, 0xAA},
		{0xF, 0xFF},
		{0x1, 0x11},
		{0x8, 0x88},
	}
	for _, tt := range tests {
		got := expand4to8(tt.input)
		if got != tt.want {
			t.Errorf("expand4to8(0x%X) = 0x%02X, want 0x%02X", tt.input, got, tt.want)
		}
	}
}

func TestRgb565ToNRGBA(t *testing.T) {
	tests := []struct {
		name  string
		input uint16
		wantR byte
		wantG byte
		wantB byte
	}{
		{"black", 0x0000, 0, 0, 0},
		{"white", 0xFFFF, 255, 255, 255},
		{"red", 0xF800, 255, 0, 0},
		{"green", 0x07E0, 0, 255, 0},
		{"blue", 0x001F, 0, 0, 255},
	}
	for _, tt := range tests {
		c := rgb565ToNRGBA(tt.input)
		if c.R != tt.wantR || c.G != tt.wantG || c.B != tt.wantB || c.A != 255 {
			t.Errorf("rgb565ToNRGBA(%s) = RGBA(%d,%d,%d,%d), want RGBA(%d,%d,%d,255)",
				tt.name, c.R, c.G, c.B, c.A, tt.wantR, tt.wantG, tt.wantB)
		}
	}
}

func TestLerpColor(t *testing.T) {
	c0 := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	c1 := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	// Midpoint: num=1, denom=1 → total=2, weight 1/2 and 1/2
	mid := lerpColor(c0, c1, 1, 1)
	if mid.R != 127 || mid.G != 127 || mid.B != 127 {
		t.Errorf("lerpColor midpoint = RGB(%d,%d,%d), want RGB(127,127,127)", mid.R, mid.G, mid.B)
	}

	// 1/3: num=1, denom=2 → total=3, weight 2/3 c0 + 1/3 c1
	third := lerpColor(c0, c1, 1, 2)
	if third.R != 85 || third.G != 85 || third.B != 85 {
		t.Errorf("lerpColor 1/3 = RGB(%d,%d,%d), want RGB(85,85,85)", third.R, third.G, third.B)
	}
}

func TestDecodeBGRA4444(t *testing.T) {
	// Layout: b1 low=B, b1 high=G, b2 low=R, b2 high=A
	data := []byte{
		0xAB, 0xCD, // B=expand(0xB)=0xBB, G=expand(0xA)=0xAA, R=expand(0xD)=0xDD, A=expand(0xC)=0xCC
		0x00, 0xFF, // B=expand(0x0)=0x00, G=expand(0x0)=0x00, R=expand(0xF)=0xFF, A=expand(0xF)=0xFF
	}
	img, _ := decodePixels(data, 2, 1, FormatBGRA4444)

	c0 := img.NRGBAAt(0, 0)
	if c0.R != 0xDD || c0.G != 0xAA || c0.B != 0xBB || c0.A != 0xCC {
		t.Errorf("pixel(0,0) = RGBA(%d,%d,%d,%d), want RGBA(0xDD,0xAA,0xBB,0xCC)", c0.R, c0.G, c0.B, c0.A)
	}

	c1 := img.NRGBAAt(1, 0)
	if c1.R != 0xFF || c1.G != 0x00 || c1.B != 0x00 || c1.A != 0xFF {
		t.Errorf("pixel(1,0) = RGBA(%d,%d,%d,%d), want RGBA(0xFF,0x00,0x00,0xFF)", c1.R, c1.G, c1.B, c1.A)
	}
}

func TestDecodeBGRA8888(t *testing.T) {
	// 2x1 image: pixel 0 = BGRA(10, 20, 30, 255), pixel 1 = BGRA(50, 60, 70, 128)
	data := []byte{
		10, 20, 30, 255, // pixel 0: B=10, G=20, R=30, A=255
		50, 60, 70, 128, // pixel 1: B=50, G=60, R=70, A=128
	}
	img, _ := decodePixels(data, 2, 1, FormatBGRA8888)

	c0 := img.NRGBAAt(0, 0)
	if c0.R != 30 || c0.G != 20 || c0.B != 10 || c0.A != 255 {
		t.Errorf("pixel(0,0) = RGBA(%d,%d,%d,%d), want RGBA(30,20,10,255)", c0.R, c0.G, c0.B, c0.A)
	}

	c1 := img.NRGBAAt(1, 0)
	if c1.R != 70 || c1.G != 60 || c1.B != 50 || c1.A != 128 {
		t.Errorf("pixel(1,0) = RGBA(%d,%d,%d,%d), want RGBA(70,60,50,128)", c1.R, c1.G, c1.B, c1.A)
	}
}

func TestDecodeBGR565(t *testing.T) {
	// Pure red: R=31, G=0, B=0 → 0xF800
	data := []byte{0x00, 0xF8}
	img, _ := decodePixels(data, 1, 1, FormatBGR565)

	c := img.NRGBAAt(0, 0)
	if c.R != 248 || c.G != 0 || c.B != 0 || c.A != 255 {
		t.Errorf("pixel = RGBA(%d,%d,%d,%d), want RGBA(248,0,0,255)", c.R, c.G, c.B, c.A)
	}
}

func TestDecodeARGB1555AlphaOn(t *testing.T) {
	// Alpha=1, R=31, G=0, B=0 → 0x8000 | 0x7C00 = 0xFC00
	data := []byte{0x00, 0xFC}
	img, _ := decodePixels(data, 1, 1, FormatARGB1555)

	c := img.NRGBAAt(0, 0)
	if c.A != 255 || c.R != 255 {
		t.Errorf("pixel = RGBA(%d,%d,%d,%d), want A=255, R=255", c.R, c.G, c.B, c.A)
	}
}

func TestDecodeARGB1555AlphaOff(t *testing.T) {
	// Alpha=0, R=31, G=0, B=0 → 0x7C00
	data := []byte{0x00, 0x7C}
	img, _ := decodePixels(data, 1, 1, FormatARGB1555)

	c := img.NRGBAAt(0, 0)
	if c.A != 0 {
		t.Errorf("pixel A = %d, want 0", c.A)
	}
}

func TestDecodeBlockRGB565(t *testing.T) {
	// Single block covers a 16x16 area with one color
	// Pure blue: R=0, G=0, B=31 → 0x001F
	data := []byte{0x1F, 0x00}
	img, _ := decodePixels(data, 16, 16, FormatBlockRGB565)

	// Check corner pixel
	c := img.NRGBAAt(0, 0)
	if c.B != 255 || c.R != 0 || c.G != 0 || c.A != 255 {
		t.Errorf("pixel(0,0) = RGBA(%d,%d,%d,%d), want RGBA(0,0,255,255)", c.R, c.G, c.B, c.A)
	}

	// Check opposite corner
	c15 := img.NRGBAAt(15, 15)
	if c15.B != 255 || c15.R != 0 {
		t.Errorf("pixel(15,15) = RGBA(%d,%d,%d,%d), want same blue", c15.R, c15.G, c15.B, c15.A)
	}
}

func TestDecodeDXT1ColorsC0GreaterThanC1(t *testing.T) {
	// c0 = 0xFFFF (white), c1 = 0x0000 (black)
	block := []byte{0xFF, 0xFF, 0x00, 0x00, 0, 0, 0, 0}
	colors := decodeDXT1Colors(block)

	if colors[0].R != 255 || colors[0].G != 255 || colors[0].B != 255 {
		t.Errorf("colors[0] = %v, want white", colors[0])
	}
	if colors[1].R != 0 || colors[1].G != 0 || colors[1].B != 0 {
		t.Errorf("colors[1] = %v, want black", colors[1])
	}
	// colors[2] should be 2/3 c0 + 1/3 c1
	if colors[2].R == 0 || colors[2].G == 0 || colors[2].B == 0 {
		t.Errorf("colors[2] = %v, want non-black interpolation", colors[2])
	}
	// colors[3] should be 1/3 c0 + 2/3 c1
	if colors[3].A != 255 {
		t.Errorf("colors[3].A = %d, want 255", colors[3].A)
	}
}

func TestDecodeDXT1ColorsC0LessOrEqualC1(t *testing.T) {
	// c0 = 0x0000 (black), c1 = 0xFFFF (white) → c0 <= c1
	block := []byte{0x00, 0x00, 0xFF, 0xFF, 0, 0, 0, 0}
	colors := decodeDXT1Colors(block)

	// colors[3] should be transparent black
	if colors[3].R != 0 || colors[3].G != 0 || colors[3].B != 0 || colors[3].A != 0 {
		t.Errorf("colors[3] = %v, want transparent black", colors[3])
	}
}

func TestIsZlibHeader(t *testing.T) {
	tests := []struct {
		header uint16
		want   bool
	}{
		{0x9C78, true},
		{0xDA78, true},
		{0x0178, true},
		{0x5E78, true},
		{0x0000, false},
		{0xFFFF, false},
		{0x1234, false},
	}
	for _, tt := range tests {
		got := isZlibHeader(tt.header)
		if got != tt.want {
			t.Errorf("isZlibHeader(0x%04X) = %v, want %v", tt.header, got, tt.want)
		}
	}
}

func TestDecompressZlib(t *testing.T) {
	original := []byte("Hello, WZ canvas decompression!")
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(original)
	w.Close()

	result, err := decompressZlib(buf.Bytes())
	if err != nil {
		t.Fatalf("decompressZlib: %v", err)
	}
	if !bytes.Equal(result, original) {
		t.Errorf("decompressZlib = %q, want %q", result, original)
	}
}

func TestDecompressEmpty(t *testing.T) {
	img, err := Decompress(nil, 4, 4, FormatBGRA8888, nil)
	if err != nil {
		t.Fatalf("Decompress(nil): %v", err)
	}
	if img.Bounds().Dx() != 4 || img.Bounds().Dy() != 4 {
		t.Errorf("bounds = %v, want 4x4", img.Bounds())
	}
}

func TestDecompressEmptySlice(t *testing.T) {
	img, err := Decompress([]byte{}, 2, 2, FormatBGRA8888, nil)
	if err != nil {
		t.Fatalf("Decompress([]byte{}): %v", err)
	}
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Errorf("bounds = %v, want 2x2", img.Bounds())
	}
}
