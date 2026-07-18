package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// openMonolithFixture builds a v12-style monolithic Data.wz (category
// subdirectories plus a root-level Base.wz sidecar image) and opens it.
func openMonolithFixture(t *testing.T) *wz.File {
	t.Helper()
	b := wztest.NewBuilder().
		SetVersion(12).
		SetEncryption(crypto.EncryptionNone).
		AddImage(wztest.Img("smap", wztest.Str("stand", "Cp"))).
		AddDir(wztest.Dir{
			Name:   "Map",
			Images: []wztest.Image{wztest.Img("000000000", wztest.Str("name", "Henesys"))},
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

// TestMonolithSubViewCategory resolves Map.wz to the Data.wz/Map subdirectory,
// so atlas-renders composites v12 maps from the monolithic archive without a
// standalone Map.wz object (task-172 C-3 — the render-side gap that made v12
// full-map renders 500 with "Map.wz: key does not exist").
func TestMonolithSubViewCategory(t *testing.T) {
	mono := openMonolithFixture(t)
	sub, err := monolithSubView(mono, "Map.wz")
	if err != nil {
		t.Fatalf("monolithSubView Map.wz: %v", err)
	}
	if sub.Name() != "Map" {
		t.Fatalf("sub name = %q, want Map", sub.Name())
	}
	imgs := sub.Root().Images()
	if len(imgs) != 1 || imgs[0].Name() != "000000000" {
		t.Fatalf("sub images = %+v, want [000000000]", imgs)
	}
}

// TestMonolithSubViewBase maps Base.wz to the Data.wz root so the root-level
// smap/zmap sidecars remain reachable.
func TestMonolithSubViewBase(t *testing.T) {
	mono := openMonolithFixture(t)
	sub, err := monolithSubView(mono, "Base.wz")
	if err != nil {
		t.Fatalf("monolithSubView Base.wz: %v", err)
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

func TestMonolithSubViewAbsentCategory(t *testing.T) {
	mono := openMonolithFixture(t)
	if _, err := monolithSubView(mono, "Sound.wz"); err == nil {
		t.Fatalf("expected error for category absent from monolithic Data.wz")
	}
}
