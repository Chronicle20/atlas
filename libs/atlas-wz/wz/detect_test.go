package wz

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// smallArchive builds a one-dir/one-image archive under the given encryption.
func smallArchive(enc crypto.EncryptionType, version int) *wztest.Builder {
	return wztest.NewBuilder().
		SetVersion(version).
		SetEncryption(enc).
		AddDir(wztest.Dir{
			Name: "Mob",
			Images: []wztest.Image{
				wztest.Img("100100", wztest.Str("name", "Snail")),
			},
		})
}

// TestDetectUnencrypted is the RC-1 regression: an unencrypted archive must
// detect EncryptionNone (empty key) and produce sane entry names — before
// this fix, detection locked in the GMS key and names decoded to garbage.
func TestDetectUnencrypted(t *testing.T) {
	path := writeFixture(t, smallArchive(crypto.EncryptionNone, 48), "Mob.wz")
	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if !f.EncryptionKey().IsEmpty() {
		t.Fatalf("expected empty (None) key, got non-empty")
	}
	if f.GameVersion() != 48 {
		t.Fatalf("game version = %d, want 48", f.GameVersion())
	}
	dirs := f.Root().Directories()
	if len(dirs) != 1 || dirs[0].Name() != "Mob" {
		t.Fatalf("root dirs = %+v, want [Mob]", dirs)
	}
	imgs := dirs[0].Images()
	if len(imgs) != 1 || imgs[0].Name() != "100100" {
		t.Fatalf("images = %+v, want [100100]", imgs)
	}
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("properties: %v", err)
	}
	if s, ok := props[0].(*property.StringProperty); !ok || s.Value() != "Snail" {
		t.Fatalf("prop = %#v, want name=Snail", props[0])
	}
}

// TestDetectGMS / TestDetectKMS: genuinely-encrypted archives still detect
// their own key (names decode sanely only under the right key).
func TestDetectGMS(t *testing.T) {
	path := writeFixture(t, smallArchive(crypto.EncryptionGMS, 83), "Mob.wz")
	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if f.EncryptionKey().IsEmpty() {
		t.Fatalf("expected GMS key, got empty")
	}
	if got := f.Root().Directories()[0].Name(); got != "Mob" {
		t.Fatalf("dir name = %q, want Mob", got)
	}
	if f.GameVersion() != 83 {
		t.Fatalf("game version = %d, want 83", f.GameVersion())
	}
}

func TestDetectKMS(t *testing.T) {
	path := writeFixture(t, smallArchive(crypto.EncryptionKMS, 185), "Mob.wz")
	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if f.EncryptionKey().IsEmpty() {
		t.Fatalf("expected KMS key, got empty")
	}
	if got := f.Root().Directories()[0].Name(); got != "Mob" {
		t.Fatalf("dir name = %q, want Mob", got)
	}
}

// TestDetectNoSaneCandidateErrors: when no key decodes the first entry name
// to something sane, Open must fail with a descriptive error — never guess.
// The raw name bytes are chosen so each candidate key decodes at least one
// byte to a control character: byte0 kills None (0xAA^mask0=0x00), byte1
// kills GMS, byte2 kills KMS.
func TestDetectNoSaneCandidateErrors(t *testing.T) {
	gms := crypto.GetKeyForRegion(crypto.EncryptionGMS).Bytes(16)
	kms := crypto.GetKeyForRegion(crypto.EncryptionKMS).Bytes(16)
	raw := []byte{0xAA, 0xAB ^ gms[1], 0xAC ^ kms[2]}
	b := smallArchive(crypto.EncryptionNone, 83).SetRawRootEntryName(raw)
	path := writeFixture(t, b, "Mob.wz")
	_, err := Open(logrus.StandardLogger(), path)
	if err == nil {
		t.Fatalf("expected detection error, got nil")
	}
	if !strings.Contains(err.Error(), "no encryption candidate") {
		t.Fatalf("error %q does not name the key-detection failure", err)
	}
}
