package wzdiff

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// TestSelfCheckCleanArchivePasses proves a well-formed archive containing a
// type-9 sub-object walks clean: zero violations, zero parse errors, and a
// non-zero SubObjects count. The SubObjects > 0 assertion is load-bearing —
// a gate that walks nothing trivially reports clean, which is exactly the
// failure mode this test exists to catch (task-262 R2).
func TestSelfCheckCleanArchivePasses(t *testing.T) {
	archive := writeArchive(t, wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionNone).
		AddDir(wztest.Dir{
			Name: "Item",
			Images: []wztest.Image{
				wztest.Img("0200",
					wztest.Sub("info",
						wztest.Int("state", 1),
						wztest.Str("name", "x"),
					),
				),
			},
		}),
		"Item.wz")

	result, err := SelfCheck(testLogger(&bytes.Buffer{}), archive)
	if err != nil {
		t.Fatalf("SelfCheck: %v", err)
	}

	if result.SubObjects == 0 {
		t.Fatalf("SubObjects = 0, want > 0 (a gate that walks nothing trivially reports clean)")
	}
	if len(result.Violations) != 0 {
		t.Errorf("Violations = %+v, want empty", result.Violations)
	}
	if len(result.ParseErrors) != 0 {
		t.Errorf("ParseErrors = %+v, want empty", result.ParseErrors)
	}
	if result.Images != 1 {
		t.Errorf("Images = %d, want 1", result.Images)
	}
}

// TestSelfCheckCorruptedArchiveFails proves that when a type-9 sub-object's
// declared size disagrees with the body actually written, SelfCheck reports
// exactly one violation naming the image, path, and both disagreeing
// offsets — the defect that image.go's recovery reseek would otherwise
// silently heal (task-262 R2).
//
// wztest.Builder has no API to express a declared-size corruption directly
// (writePropList always writes inner.Len() as the true length), so this
// test patches the serialized int32 length prefix directly in the built
// bytes: locate it via a clean trace pass (the "sub" event's StartOff is
// exactly the length prefix's file offset — parsePropertyValue captures
// `start` right after the type byte, and the type-9 branch reads the
// length as the very next 4 bytes), then overwrite it with a wrong value.
func TestSelfCheckCorruptedArchiveFails(t *testing.T) {
	builder := wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionNone).
		AddDir(wztest.Dir{
			Name: "Item",
			Images: []wztest.Image{
				wztest.Img("0200",
					wztest.Sub("info",
						wztest.Int("state", 1),
					),
				),
			},
		})

	data, err := builder.Build()
	if err != nil {
		t.Fatalf("build archive: %v", err)
	}

	cleanPath := writeBytesArchive(t, data, "ItemClean.wz")
	subEvent := traceSubEvent(t, cleanPath)

	// The length prefix is the 4 bytes immediately preceding subEvent's
	// StartOff (parsePropertyValue's "start" is captured right after the
	// type-9 tag byte, and the type-9 branch's very next read is the int32
	// length). Overwrite it with a value one byte short of the true
	// declared size, so the decode's ActualEnd disagrees with the
	// corrupted DeclaredEnd.
	sizeFieldStart := subEvent.StartOff
	corrupted := append([]byte(nil), data...)
	trueSize := binary.LittleEndian.Uint32(corrupted[sizeFieldStart : sizeFieldStart+4])
	binary.LittleEndian.PutUint32(corrupted[sizeFieldStart:sizeFieldStart+4], trueSize-1)

	corruptedPath := writeBytesArchive(t, corrupted, "ItemCorrupted.wz")

	result, err := SelfCheck(testLogger(&bytes.Buffer{}), corruptedPath)
	if err != nil {
		t.Fatalf("SelfCheck: %v", err)
	}

	if len(result.Violations) != 1 {
		t.Fatalf("Violations = %+v, want exactly 1", result.Violations)
	}
	v := result.Violations[0]
	if v.Image != "0200" {
		t.Errorf("Violations[0].Image = %q, want %q", v.Image, "0200")
	}
	if v.Path != "/info" {
		t.Errorf("Violations[0].Path = %q, want %q", v.Path, "/info")
	}
	wantDeclaredEnd := subEvent.DeclaredEnd - 1
	if v.DeclaredEnd != wantDeclaredEnd {
		t.Errorf("Violations[0].DeclaredEnd = %d, want %d", v.DeclaredEnd, wantDeclaredEnd)
	}
	if v.ActualEnd != subEvent.ActualEnd {
		t.Errorf("Violations[0].ActualEnd = %d, want %d (the decode itself is unaffected by the corruption)", v.ActualEnd, subEvent.ActualEnd)
	}
}

// writeBytesArchive writes already-built archive bytes to a temp file,
// mirroring writeArchive's file-writing half without re-invoking Build.
func writeBytesArchive(t *testing.T, data []byte, name string) string {
	t.Helper()
	path := t.TempDir() + "/" + name
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	return path
}

// traceSubEvent opens path, installs a trace hook, parses every image, and
// returns the sole Kind == "sub" event observed. Used to locate the
// declared-size field's exact file offset without hard-coding the wztest
// serialization layout in the test.
func traceSubEvent(t *testing.T, path string) wz.TraceEvent {
	t.Helper()
	f, err := wz.Open(testLogger(&bytes.Buffer{}), path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()

	images := map[string]*wz.Image{}
	collectImages(f.Root(), images)

	var found *wz.TraceEvent
	f.SetTrace(func(ev wz.TraceEvent) {
		if ev.Kind != "sub" {
			return
		}
		found = &ev
	})

	for _, img := range images {
		if _, err := img.Properties(); err != nil {
			t.Fatalf("parse image: %v", err)
		}
	}

	if found == nil {
		t.Fatalf("no sub event observed while tracing %s", path)
	}
	return *found
}
