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

// manyImageArchive builds a Mob-style archive with n image entries at the given
// version — mirrors the real GMS Mob.wz shape (a flat directory of mob imgs).
func manyImageArchive(enc crypto.EncryptionType, version, n int) *wztest.Builder {
	imgs := make([]wztest.Image, 0, n)
	for i := 0; i < n; i++ {
		id := 100100 + i
		imgs = append(imgs, wztest.Img(strconvItoa(id), wztest.Str("name", "Mob")))
	}
	return wztest.NewBuilder().
		SetVersion(version).
		SetEncryption(enc).
		AddDir(wztest.Dir{Name: "Mob", Images: imgs})
}

func strconvItoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}

// TestDetectManyEntryArchive verifies that the strengthened version detection —
// which now validates EVERY root directory entry's offset, not just the first —
// still detects the correct version for a many-entry archive and does not
// over-reject a legitimate one. This is the direct guard for the GMS v72 Mob.wz
// fix: the ~8-bit encrypted-version header collides across versions (v72 with
// v3), and first-entry-only validation let the wrong (lower) version win, whose
// wrong hash miscomputed every image offset → EOF on every monster image → zero
// monsters ingested. The full failure only reproduces on a multi-megabyte
// archive (a wrong hash's first offset must land in-bounds by chance), which is
// impractical to synthesize here; the root cause is confirmed from the ingest
// log ("Detected version 3"). This test locks in that validating all 64 entries
// does not break correct detection. A v72 archive must detect exactly v72 and
// every image must parse.
func TestDetectManyEntryArchive(t *testing.T) {
	path := writeFixture(t, manyImageArchive(crypto.EncryptionGMS, 72, 64), "Mob.wz")
	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if f.GameVersion() != 72 {
		t.Fatalf("game version = %d, want 72 (mis-detection = weak version validation)", f.GameVersion())
	}
	imgs := f.Root().Directories()[0].Images()
	if len(imgs) != 64 {
		t.Fatalf("images = %d, want 64", len(imgs))
	}
	for _, img := range imgs {
		if _, err := img.Properties(); err != nil {
			t.Fatalf("image [%s] failed to parse (wrong offsets): %v", img.Name(), err)
		}
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
