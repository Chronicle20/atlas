package wzdiff

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

func testLogger(buf *bytes.Buffer) *logrus.Logger {
	l := logrus.New()
	l.Out = buf
	l.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	return l
}

func writeArchive(t *testing.T, b *wztest.Builder, name string) string {
	t.Helper()
	data, err := b.Build()
	if err != nil {
		t.Fatalf("build archive: %v", err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	return path
}

// TestRunReportsImageSetMismatch proves the "419 vs 421" image-set gap
// surfaces as its own number, exactly as
// evidence-wz-parse-divergence-reactor.txt:1-3 does, and that the missing
// image is named for a human to go look at.
func TestRunReportsImageSetMismatch(t *testing.T) {
	archive := writeArchive(t, wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionNone).
		AddImage(wztest.Img("2000", wztest.Sub("info", wztest.Int("state", 1)))),
		"Reactor.wz")

	refDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(refDir, "2000.img.xml"),
		[]byte(`<imgdir name="2000.img"><imgdir name="info"><int name="state" value="1"/></imgdir></imgdir>`), 0o644); err != nil {
		t.Fatalf("write reference xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refDir, "2001.img.xml"),
		[]byte(`<imgdir name="2001.img"></imgdir>`), 0o644); err != nil {
		t.Fatalf("write reference xml: %v", err)
	}

	var logBuf bytes.Buffer
	result, err := Run(testLogger(&logBuf), archive, refDir, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.ImagesOurs != 1 {
		t.Errorf("ImagesOurs = %d, want 1", result.ImagesOurs)
	}
	if result.ImagesReference != 2 {
		t.Errorf("ImagesReference = %d, want 2", result.ImagesReference)
	}
	if !strings.Contains(logBuf.String(), "2001") {
		t.Errorf("log output does not name the missing image 2001: %q", logBuf.String())
	}
	if len(result.OnlyOurs) != 0 {
		t.Errorf("OnlyOurs = %v, want empty", result.OnlyOurs)
	}
	if want := []string{"2001"}; !equalStrings(result.OnlyReference, want) {
		t.Errorf("OnlyReference = %v, want %v", result.OnlyReference, want)
	}
}

// TestRunReportsImageSetMismatchWithEqualCounts proves that a same-count,
// different-membership image set is caught: ImagesOurs == ImagesReference
// alone must not be mistaken for the sets being equal.
func TestRunReportsImageSetMismatchWithEqualCounts(t *testing.T) {
	archive := writeArchive(t, wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionNone).
		AddImage(wztest.Img("2000", wztest.Sub("info", wztest.Int("state", 1)))).
		AddImage(wztest.Img("2001", wztest.Sub("info", wztest.Int("state", 1)))),
		"Reactor.wz")

	refDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(refDir, "2000.img.xml"),
		[]byte(`<imgdir name="2000.img"><imgdir name="info"><int name="state" value="1"/></imgdir></imgdir>`), 0o644); err != nil {
		t.Fatalf("write reference xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refDir, "2002.img.xml"),
		[]byte(`<imgdir name="2002.img"></imgdir>`), 0o644); err != nil {
		t.Fatalf("write reference xml: %v", err)
	}

	var logBuf bytes.Buffer
	result, err := Run(testLogger(&logBuf), archive, refDir, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.ImagesOurs != result.ImagesReference {
		t.Fatalf("ImagesOurs/ImagesReference = %d/%d, want equal counts for this test", result.ImagesOurs, result.ImagesReference)
	}
	if want := []string{"2001"}; !equalStrings(result.OnlyOurs, want) {
		t.Errorf("OnlyOurs = %v, want %v", result.OnlyOurs, want)
	}
	if want := []string{"2002"}; !equalStrings(result.OnlyReference, want) {
		t.Errorf("OnlyReference = %v, want %v", result.OnlyReference, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRunCleanArchiveHasNoDeltas proves a matching archive/dump pair
// produces zero deltas.
func TestRunCleanArchiveHasNoDeltas(t *testing.T) {
	archive := writeArchive(t, wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionNone).
		AddImage(wztest.Img("2000", wztest.Sub("info", wztest.Int("state", 1)))),
		"Reactor.wz")

	refDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(refDir, "2000.img.xml"),
		[]byte(`<imgdir name="2000.img"><imgdir name="info"><int name="state" value="1"/></imgdir></imgdir>`), 0o644); err != nil {
		t.Fatalf("write reference xml: %v", err)
	}

	var logBuf bytes.Buffer
	result, err := Run(testLogger(&logBuf), archive, refDir, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.ImagesOurs != 1 || result.ImagesReference != 1 {
		t.Fatalf("ImagesOurs/ImagesReference = %d/%d, want 1/1", result.ImagesOurs, result.ImagesReference)
	}
	if len(result.Divergent) != 0 {
		t.Errorf("Divergent = %+v, want empty", result.Divergent)
	}
}
