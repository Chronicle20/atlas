package workers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// openMonolithFixture builds a v12-style Data.wz on disk and opens it.
func openMonolithFixture(t *testing.T) *wz.File {
	t.Helper()
	b := wztest.NewBuilder().
		SetVersion(12).
		SetEncryption(crypto.EncryptionNone).
		AddImage(wztest.Img("smap", wztest.Str("stand", "Cp"))).
		AddDir(wztest.Dir{
			Name:   "Item",
			Images: []wztest.Image{wztest.Img("0200", wztest.Str("name", "Red Potion"))},
		}).
		AddDir(wztest.Dir{
			Name:   "Mob",
			Images: []wztest.Image{wztest.Img("100100", wztest.Str("name", "Snail"))},
		})
	data, err := b.Build()
	if err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "Data.wz")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	f, err := wz.Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(f.Close)
	return f
}

func TestMonolithSubArchiveCategory(t *testing.T) {
	mono := openMonolithFixture(t)
	sub, err := monolithSubArchive(mono, "Item.wz")
	if err != nil {
		t.Fatalf("monolithSubArchive Item.wz: %v", err)
	}
	if sub.Name() != "Item" {
		t.Fatalf("sub name = %q, want Item", sub.Name())
	}
	imgs := sub.Root().Images()
	if len(imgs) != 1 || imgs[0].Name() != "0200" {
		t.Fatalf("sub images = %+v, want [0200]", imgs)
	}
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("sub image parse: %v", err)
	}
	if s := props[0].(*property.StringProperty); s.Value() != "Red Potion" {
		t.Fatalf("prop = %q, want Red Potion", s.Value())
	}
}

// TestMonolithSubArchiveBase: Base.wz resolves to the Data.wz ROOT so the
// character worker's smap/zmap sidecars read the root-level images.
func TestMonolithSubArchiveBase(t *testing.T) {
	mono := openMonolithFixture(t)
	sub, err := monolithSubArchive(mono, "Base.wz")
	if err != nil {
		t.Fatalf("monolithSubArchive Base.wz: %v", err)
	}
	found := false
	for _, img := range sub.Root().Images() {
		if img.Name() == "smap" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Base.wz view must expose root-level smap image")
	}
}

func TestMonolithSubArchiveAbsentCategory(t *testing.T) {
	mono := openMonolithFixture(t)
	_, err := monolithSubArchive(mono, "Quest.wz")
	if !errors.Is(err, ErrCategoryAbsent) {
		t.Fatalf("err = %v, want ErrCategoryAbsent", err)
	}
}

func TestOpenArchiveNilClient(t *testing.T) {
	_, _, err := OpenArchive(context.Background(), logrus.StandardLogger(), nil, Params{}, "Item.wz")
	if err == nil {
		t.Fatalf("expected error for nil minio client")
	}
}
