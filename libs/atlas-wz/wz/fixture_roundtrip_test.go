package wz

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
	"github.com/sirupsen/logrus"
)

// writeFixture materializes builder output as a .wz file in a temp dir.
func writeFixture(t *testing.T, b *wztest.Builder, name string) string {
	t.Helper()
	data, err := b.Build()
	if err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func findProp(t *testing.T, props []property.Property, name string) property.Property {
	t.Helper()
	for _, p := range props {
		if p.Name() == name {
			return p
		}
	}
	t.Fatalf("property %q not found", name)
	return nil
}

// TestFixtureRoundTripGMS proves the builder emits archives the CURRENT
// parser accepts: GMS-encrypted archives are the already-working path, so
// this round-trip is valid before any detection change lands.
func TestFixtureRoundTripGMS(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	b := wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionGMS).
		AddDir(wztest.Dir{
			Name: "Consume",
			Images: []wztest.Image{
				wztest.Img("0200",
					wztest.Sub("02000000",
						wztest.Str("name", "Red Potion"),
						wztest.Int("price", 50),
					),
					wztest.Canvas("icon", payload),
				),
			},
		})
	path := writeFixture(t, b, "Item.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	if f.Name() != "Item" {
		t.Fatalf("name = %q, want Item", f.Name())
	}
	dirs := f.Root().Directories()
	if len(dirs) != 1 || dirs[0].Name() != "Consume" {
		t.Fatalf("root dirs = %+v, want one dir Consume", dirs)
	}
	imgs := dirs[0].Images()
	if len(imgs) != 1 || imgs[0].Name() != "0200" {
		t.Fatalf("images = %+v, want one image 0200", imgs)
	}
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("properties: %v", err)
	}
	sub, ok := findProp(t, props, "02000000").(*property.SubProperty)
	if !ok {
		t.Fatalf("02000000 is not a SubProperty")
	}
	if s, ok := findProp(t, sub.Children(), "name").(*property.StringProperty); !ok || s.Value() != "Red Potion" {
		t.Fatalf("name prop = %#v, want Red Potion", findProp(t, sub.Children(), "name"))
	}
	if iv, ok := findProp(t, sub.Children(), "price").(*property.IntProperty); !ok || iv.Value() != 50 {
		t.Fatalf("price prop = %#v, want 50", findProp(t, sub.Children(), "price"))
	}
	cp, ok := findProp(t, props, "icon").(*property.CanvasProperty)
	if !ok {
		t.Fatalf("icon is not a CanvasProperty")
	}
	got, err := f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		t.Fatalf("read canvas: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("canvas payload = %v, want %v", got, payload)
	}
}
