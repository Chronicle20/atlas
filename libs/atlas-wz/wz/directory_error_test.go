package wz

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// The constants below mirror the fixed encoding lengths wztest.Builder.Build
// produces for a two-level archive (root -> "Outer" dir -> "Inner" dir ->
// "0200" image with a single "price" Int property), so the entry-count byte
// of the *nested* ("Inner") directory chunk can be located and corrupted in
// the built bytes without any knowledge of the archive's encryption key:
// every WZ string/int encoding's *byte length* is independent of the key
// (XOR does not change length), so it can be computed purely from the
// structure below.
const (
	fixtureDesc      = "Package file test" // wztest.Builder.Build's fixed PKG1 description
	fixtureImgName   = "0200"
	fixturePropName  = "price"
	fixturePropValue = int32(50)
	fixtureInnerName = "Inner"
	fixtureOuterName = "Outer"
)

// wzIntSize returns the number of bytes wztest's writeWzInt (== the reader's
// ReadWzInt encoding) uses for v: 1 byte for values in [-127,127], else 5.
func wzIntSize(v int32) int {
	if v >= -127 && v <= 127 {
		return 1
	}
	return 5
}

// wzStringSize is the byte length of a bare WZ string (tag byte + payload),
// as written directly by wztest's writeWzString - used for directory entry
// names.
func wzStringSize(s string) int { return 1 + len(s) }

// stringBlockSize is the byte length of a WZ string *block* (the extra 0x73
// tag byte plus a bare WZ string) - used for image property names/tags.
func stringBlockSize(s string) int { return 1 + wzStringSize(s) }

// nestedArchiveFixture builds the two-level archive described above and
// returns its bytes plus the absolute byte offset of the nested ("Inner")
// directory's own entry-count byte.
func nestedArchiveFixture(t *testing.T) (data []byte, innerCountOffset int) {
	t.Helper()

	b := wztest.NewBuilder().
		SetVersion(83).
		AddDir(wztest.Dir{
			Name: fixtureOuterName,
			Dirs: []wztest.Dir{
				{
					Name: fixtureInnerName,
					Images: []wztest.Image{
						wztest.Img(fixtureImgName, wztest.Int(fixturePropName, fixturePropValue)),
					},
				},
			},
		})

	data, err := b.Build()
	if err != nil {
		t.Fatalf("build fixture: %v", err)
	}

	contentStart := 4 + 8 + 4 + len(fixtureDesc) + 1

	imgSize := stringBlockSize("Property") + 2 + // "Property" tag + hasProperty(0,0)
		wzIntSize(1) + // property count
		stringBlockSize(fixturePropName) + 1 /* Int type byte */ + wzIntSize(fixturePropValue)

	innerEntrySize := 1 /* type */ + wzStringSize(fixtureImgName+".img") +
		wzIntSize(int32(imgSize)) + wzIntSize(0) /* checksum */ + 4 /* offset */
	innerDirSize := wzIntSize(1) + innerEntrySize

	outerEntrySize := 1 + wzStringSize(fixtureInnerName) + wzIntSize(int32(innerDirSize)) + wzIntSize(0) + 4
	outerDirSize := wzIntSize(1) + outerEntrySize

	rootEntrySize := 1 + wzStringSize(fixtureOuterName) + wzIntSize(int32(outerDirSize)) + wzIntSize(0) + 4
	rootDirSize := wzIntSize(1) + rootEntrySize

	// Layout: header(contentStart) + ev(2) + root chunk + image chunk + inner
	// dir chunk + outer dir chunk (chunks are appended pre-root in the order
	// they were built, i.e. innermost-first: image, inner dir, outer dir).
	innerCountOffset = contentStart + 2 + rootDirSize + imgSize

	return data, innerCountOffset
}

func writeArchive(t *testing.T, data []byte, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// TestSubDirectoryParseFailurePropagates asserts that a nested sub-directory
// whose entry-count byte is corrupted causes wz.Open to fail hard, naming
// the failing sub-directory, instead of silently dropping it and every image
// beneath it.
func TestSubDirectoryParseFailurePropagates(t *testing.T) {
	data, innerCountOffset := nestedArchiveFixture(t)

	// Sanity-check the computed offset actually lands on the byte we expect
	// (the single-byte encoded entry count "1" for Inner's one image).
	if data[innerCountOffset] != 1 {
		t.Fatalf("computed innerCountOffset=%d does not point at Inner's entry count (got %d, want 1)", innerCountOffset, data[innerCountOffset])
	}

	// Corrupt: claim 127 entries where only one exists, forcing the parser
	// to read far past the chunk's actual data.
	corrupted := append([]byte(nil), data...)
	corrupted[innerCountOffset] = 0x7F

	path := writeArchive(t, corrupted, "Nested.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err == nil {
		f.Close()
		t.Fatalf("Open succeeded on a corrupted nested directory, want error")
	}
	if f != nil {
		t.Fatalf("Open returned a non-nil *File alongside an error: %+v", f)
	}
	if !strings.Contains(err.Error(), fixtureInnerName) {
		t.Fatalf("error %q does not name the failing sub-directory %q", err.Error(), fixtureInnerName)
	}
}

// TestValidNestedArchiveStillOpens asserts the same archive, uncorrupted,
// opens cleanly and enumerates every image beneath the nested directory.
func TestValidNestedArchiveStillOpens(t *testing.T) {
	data, _ := nestedArchiveFixture(t)
	path := writeArchive(t, data, "Nested.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	outerDirs := f.Root().Directories()
	if len(outerDirs) != 1 || outerDirs[0].Name() != fixtureOuterName {
		t.Fatalf("root dirs = %+v, want one dir %q", outerDirs, fixtureOuterName)
	}

	innerDirs := outerDirs[0].Directories()
	if len(innerDirs) != 1 || innerDirs[0].Name() != fixtureInnerName {
		t.Fatalf("outer dirs = %+v, want one dir %q", innerDirs, fixtureInnerName)
	}

	imgs := innerDirs[0].Images()
	if len(imgs) != 1 || imgs[0].Name() != fixtureImgName {
		t.Fatalf("inner images = %+v, want one image %q", imgs, fixtureImgName)
	}
}
