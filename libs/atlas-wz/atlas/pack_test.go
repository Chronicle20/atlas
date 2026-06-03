package atlas

import (
	"bytes"
	"image"
	"image/color"
	"math/rand"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/manifest"
	"github.com/Chronicle20/atlas/libs/atlas-wz/atlas/pngenc"
)

// makeSprites builds a deterministic 200-sprite fixture of varied sizes.
func makeSprites(seed int64) []Input {
	rng := rand.New(rand.NewSource(seed))
	out := make([]Input, 200)
	for i := 0; i < 200; i++ {
		w := 4 + rng.Intn(32)
		h := 4 + rng.Intn(32)
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.Set(x, y, color.NRGBA{R: uint8(i), G: uint8(x), B: uint8(y), A: 255})
			}
		}
		out[i] = Input{
			Name: nameOf(i),
			Img:  img,
			Origin: image.Point{X: w / 2, Y: h / 2},
			Anchors: map[string]image.Point{
				"neck": {X: w / 4, Y: h / 4},
			},
			Z: nameOf(i),
		}
	}
	return out
}

func nameOf(i int) string {
	return string([]byte{byte('a' + i/26%26), byte('a' + i%26)})
}

func encodeSheet(t *testing.T, sheet image.Image) []byte {
	var buf bytes.Buffer
	if err := pngenc.Encode(&buf, sheet); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestPackByteIdenticalAcrossRuns(t *testing.T) {
	in := makeSprites(42)
	sheetA, manA, err := Pack(in)
	if err != nil {
		t.Fatal(err)
	}
	sheetB, manB, err := Pack(in)
	if err != nil {
		t.Fatal(err)
	}
	bytesA := encodeSheet(t, sheetA)
	bytesB := encodeSheet(t, sheetB)
	if !bytes.Equal(bytesA, bytesB) {
		t.Fatalf("sheet bytes differ across runs: %d vs %d", len(bytesA), len(bytesB))
	}
	mA, _ := manifest.Marshal(manA)
	mB, _ := manifest.Marshal(manB)
	if !bytes.Equal(mA, mB) {
		t.Fatalf("manifest bytes differ across runs:\n A=%s\n B=%s", mA, mB)
	}
}
