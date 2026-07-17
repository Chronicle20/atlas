package wz

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
	"github.com/sirupsen/logrus"
)

// monolithicArchive mimics GMS v12 Data.wz: category subdirectories plus
// root-level smap/zmap images (the Base.wz role).
func monolithicArchive(payload []byte) *wztest.Builder {
	return wztest.NewBuilder().
		SetVersion(12).
		SetEncryption(crypto.EncryptionNone).
		AddImage(wztest.Img("smap", wztest.Str("stand", "Cp"))).
		AddImage(wztest.Img("zmap", wztest.Str("dummy", "x"))).
		AddDir(wztest.Dir{
			Name: "Item",
			Images: []wztest.Image{
				wztest.Img("0200",
					wztest.Sub("02000000", wztest.Str("name", "Red Potion")),
					wztest.Canvas("icon", payload),
				),
			},
		}).
		AddDir(wztest.Dir{
			Name:   "Mob",
			Images: []wztest.Image{wztest.Img("100100", wztest.Str("name", "Snail"))},
		})
}

func TestNewSubFile(t *testing.T) {
	payload := []byte{7, 7, 7, 7}
	path := writeFixture(t, monolithicArchive(payload), "Data.wz")
	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	var itemDir *Directory
	for _, d := range f.Root().Directories() {
		if d.Name() == "Item" {
			itemDir = d
		}
	}
	if itemDir == nil {
		t.Fatalf("Item subdirectory not found")
	}

	sub := NewSubFile(f, itemDir, "Item")
	if sub.Name() != "Item" {
		t.Fatalf("sub name = %q, want Item", sub.Name())
	}
	if sub.GameVersion() != f.GameVersion() {
		t.Fatalf("sub game version %d != parent %d", sub.GameVersion(), f.GameVersion())
	}
	imgs := sub.Root().Images()
	if len(imgs) != 1 || imgs[0].Name() != "0200" {
		t.Fatalf("sub images = %+v, want [0200]", imgs)
	}
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("sub image properties: %v", err)
	}
	var cp *property.CanvasProperty
	for _, p := range props {
		if c, ok := p.(*property.CanvasProperty); ok {
			cp = c
		}
	}
	if cp == nil {
		t.Fatalf("canvas not found in sub image")
	}
	got, err := sub.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		t.Fatalf("sub ReadCanvasData: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("canvas payload = %v, want %v", got, payload)
	}

	// Close on the sub-file must NOT close the shared handle: the parent
	// (and other sub-views) must still be readable afterwards.
	sub.Close()
	for _, d := range f.Root().Directories() {
		if d.Name() != "Mob" {
			continue
		}
		mprops, err := d.Images()[0].Properties()
		if err != nil {
			t.Fatalf("parent read after sub.Close: %v", err)
		}
		if s := mprops[0].(*property.StringProperty); s.Value() != "Snail" {
			t.Fatalf("mob name = %q, want Snail", s.Value())
		}
	}
}
