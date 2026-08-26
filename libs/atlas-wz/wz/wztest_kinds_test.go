package wz

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// TestBuilderEmitsAllPropertyKinds round-trips every wztest.Prop kind
// through the real parser, proving the builder's byte encodings match
// image.go's parsePropertyValue/parseExtendedProperty exactly.
func TestBuilderEmitsAllPropertyKinds(t *testing.T) {
	b := wztest.NewBuilder().SetVersion(83).SetEncryption(crypto.EncryptionGMS).
		AddImage(wztest.Img("kinds",
			wztest.Null("n"),
			wztest.Short("s", -3),
			wztest.Int("i", 42),
			wztest.Long("l", 9000000000),
			wztest.Float("f", 1.5),
			wztest.Double("d", 2.25),
			wztest.Str("str", "hello"),
			wztest.Vector("lt", -100, -100),
			wztest.UOL("u", "../0/0"),
			wztest.Convex("cv", wztest.Vector("0", 1, 2), wztest.Vector("1", 3, 4)),
		))
	path := writeFixture(t, b, "Kinds.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	imgs := f.Root().Images()
	if len(imgs) != 1 {
		t.Fatalf("images = %+v, want one image", imgs)
	}
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("properties: %v", err)
	}

	if _, ok := findProp(t, props, "n").(*property.NullProperty); !ok {
		t.Fatalf("n is not a NullProperty: %#v", findProp(t, props, "n"))
	}

	if sp, ok := findProp(t, props, "s").(*property.ShortProperty); !ok || sp.Value() != int16(-3) {
		t.Fatalf("s prop = %#v, want -3", findProp(t, props, "s"))
	}

	if ip, ok := findProp(t, props, "i").(*property.IntProperty); !ok || ip.Value() != int32(42) {
		t.Fatalf("i prop = %#v, want 42", findProp(t, props, "i"))
	}

	if lp, ok := findProp(t, props, "l").(*property.LongProperty); !ok || lp.Value() != int64(9000000000) {
		t.Fatalf("l prop = %#v, want 9000000000", findProp(t, props, "l"))
	}

	if fp, ok := findProp(t, props, "f").(*property.FloatProperty); !ok || fp.Value() != float32(1.5) {
		t.Fatalf("f prop = %#v, want 1.5", findProp(t, props, "f"))
	}

	if dp, ok := findProp(t, props, "d").(*property.DoubleProperty); !ok || dp.Value() != float64(2.25) {
		t.Fatalf("d prop = %#v, want 2.25", findProp(t, props, "d"))
	}

	if strp, ok := findProp(t, props, "str").(*property.StringProperty); !ok || strp.Value() != "hello" {
		t.Fatalf("str prop = %#v, want hello", findProp(t, props, "str"))
	}

	if vp, ok := findProp(t, props, "lt").(*property.VectorProperty); !ok || vp.X() != -100 || vp.Y() != -100 {
		t.Fatalf("lt prop = %#v, want (-100,-100)", findProp(t, props, "lt"))
	}

	if up, ok := findProp(t, props, "u").(*property.UOLProperty); !ok || up.Value() != "../0/0" {
		t.Fatalf("u prop = %#v, want ../0/0", findProp(t, props, "u"))
	}

	cv, ok := findProp(t, props, "cv").(*property.ConvexProperty)
	if !ok {
		t.Fatalf("cv is not a ConvexProperty: %#v", findProp(t, props, "cv"))
	}
	children := cv.Children()
	if len(children) != 2 {
		t.Fatalf("cv children = %+v, want 2", children)
	}
	c0, ok := children[0].(*property.VectorProperty)
	if !ok || c0.Name() != "0" || c0.X() != 1 || c0.Y() != 2 {
		t.Fatalf("cv child 0 = %#v, want name=0 (1,2)", children[0])
	}
	c1, ok := children[1].(*property.VectorProperty)
	if !ok || c1.Name() != "1" || c1.X() != 3 || c1.Y() != 4 {
		t.Fatalf("cv child 1 = %#v, want name=1 (3,4)", children[1])
	}
}

// TestBuilderFloatZeroWithoutMarker proves the writer's fixed 0x80 marker
// byte round-trips a zero float correctly (the marker only matters for
// distinguishing "reads as zero" from "reads the real value").
func TestBuilderFloatZeroWithoutMarker(t *testing.T) {
	b := wztest.NewBuilder().SetVersion(83).SetEncryption(crypto.EncryptionGMS).
		AddImage(wztest.Img("zero", wztest.Float("f", 0)))
	path := writeFixture(t, b, "Zero.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	imgs := f.Root().Images()
	if len(imgs) != 1 {
		t.Fatalf("images = %+v, want one image", imgs)
	}
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("properties: %v", err)
	}

	fp, ok := findProp(t, props, "f").(*property.FloatProperty)
	if !ok || fp.Value() != 0 {
		t.Fatalf("f prop = %#v, want 0", findProp(t, props, "f"))
	}
}
