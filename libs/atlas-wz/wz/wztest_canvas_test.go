package wz

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// TestBuilderCanvasWithDimensionsAndChildren proves wztest.CanvasWith emits a
// canvas whose real width, height, and child property list the CURRENT
// parser (parseCanvasProperty, image.go) actually reads back, and that the
// dataOffset/dataSize it records still round-trips through
// File.ReadCanvasData with the 0xAB flag-byte pairing intact.
func TestBuilderCanvasWithDimensionsAndChildren(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	b := wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionGMS).
		AddImage(wztest.Img("frames",
			wztest.Sub("1",
				wztest.CanvasWith("0", 100, 121, payload,
					wztest.Vector("origin", 49, 121),
					wztest.Int("z", 0),
					wztest.Int("delay", 150),
				),
			),
		))
	path := writeFixture(t, b, "Frames.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	imgs := f.Root().Images()
	if len(imgs) != 1 || imgs[0].Name() != "frames" {
		t.Fatalf("images = %+v, want one image frames", imgs)
	}
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("properties: %v", err)
	}
	sub, ok := findProp(t, props, "1").(*property.SubProperty)
	if !ok {
		t.Fatalf("1 is not a SubProperty")
	}
	cp, ok := findProp(t, sub.Children(), "0").(*property.CanvasProperty)
	if !ok {
		t.Fatalf("0 is not a CanvasProperty")
	}
	if cp.Width() != 100 {
		t.Fatalf("Width() = %d, want 100", cp.Width())
	}
	if cp.Height() != 121 {
		t.Fatalf("Height() = %d, want 121", cp.Height())
	}
	if len(cp.Children()) != 3 {
		t.Fatalf("len(Children()) = %d, want 3", len(cp.Children()))
	}
	origin, ok := findProp(t, cp.Children(), "origin").(*property.VectorProperty)
	if !ok {
		t.Fatalf("origin is not a VectorProperty")
	}
	if origin.X() != 49 || origin.Y() != 121 {
		t.Fatalf("origin = (%d,%d), want (49,121)", origin.X(), origin.Y())
	}
	z, ok := findProp(t, cp.Children(), "z").(*property.IntProperty)
	if !ok {
		t.Fatalf("z is not an IntProperty")
	}
	if z.Value() != 0 {
		t.Fatalf("z = %d, want 0", z.Value())
	}
	delay, ok := findProp(t, cp.Children(), "delay").(*property.IntProperty)
	if !ok {
		t.Fatalf("delay is not an IntProperty")
	}
	if delay.Value() != 150 {
		t.Fatalf("delay = %d, want 150", delay.Value())
	}
	got, err := f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		t.Fatalf("read canvas: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("canvas payload = %v, want %v", got, payload)
	}
}

// TestBuilderCanvasBackCompat proves wztest.Canvas is still exactly the
// 1x1, no-children, additive-only wrapper TestFixtureRoundTripGMS depends
// on: CanvasWith(name, 1, 1, payload) with no children.
func TestBuilderCanvasBackCompat(t *testing.T) {
	payload := []byte{0xAA, 0xBB, 0xCC}
	b := wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionGMS).
		AddImage(wztest.Img("icon", wztest.Canvas("icon", payload)))
	path := writeFixture(t, b, "Icon.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	imgs := f.Root().Images()
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("properties: %v", err)
	}
	cp, ok := findProp(t, props, "icon").(*property.CanvasProperty)
	if !ok {
		t.Fatalf("icon is not a CanvasProperty")
	}
	if cp.Width() != 1 {
		t.Fatalf("Width() = %d, want 1", cp.Width())
	}
	if cp.Height() != 1 {
		t.Fatalf("Height() = %d, want 1", cp.Height())
	}
	if len(cp.Children()) != 0 {
		t.Fatalf("len(Children()) = %d, want 0", len(cp.Children()))
	}
	got, err := f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		t.Fatalf("read canvas: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("canvas payload = %v, want %v", got, payload)
	}
}
