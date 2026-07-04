package baseline

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	minio "atlas-data/storage/minio"

	"github.com/sirupsen/logrus"
)

func TestParseDumpKeyRoundTrip(t *testing.T) {
	cases := []struct {
		region string
		major  int
		minor  int
	}{
		{"GMS", 83, 1},
		{"GMS", 84, 1},
		{"JMS", 185, 1},
	}
	for _, c := range cases {
		key := DumpKey(c.region, c.major, c.minor)
		region, major, minor, ok := parseDumpKey(key)
		if !ok {
			t.Fatalf("parseDumpKey(%q) not ok", key)
		}
		if region != c.region || major != c.major || minor != c.minor {
			t.Fatalf("parseDumpKey(%q) = %s/%d.%d", key, region, major, minor)
		}
	}
}

func TestParseDumpKeyRejectsMalformed(t *testing.T) {
	bad := []string{
		"",
		"baseline/regions/GMS/versions/83.1/other.file",
		"baseline/regions/GMS/versions/83.1",
		"baseline/regions/GMS/versions/831/documents.dump",
		"baseline/regions/GMS/versions/x.y/documents.dump",
		"baseline/regions/GMS/versions/83./documents.dump",
		"baseline/regions/GMS/versions/.1/documents.dump",
		"baseline/regions/GMS/versions/-1.2/documents.dump",
		"baseline/regions/GMS/versions/83.-2/documents.dump",
		"baseline/regions//versions/83.1/documents.dump",
		"shared/regions/GMS/versions/83.1/documents.dump",
		"baseline/other/GMS/versions/83.1/documents.dump",
		"baseline/regions/GMS/versions/83.1/extra/documents.dump",
	}
	for _, key := range bad {
		if _, _, _, ok := parseDumpKey(key); ok {
			t.Fatalf("parseDumpKey(%q) unexpectedly ok", key)
		}
	}
}

// fakeStore is the map-backed objectStore used to drive Lister without a
// live MinIO.
type fakeStore struct {
	objs    []minio.ObjectInfo
	blobs   map[string][]byte
	listErr error
}

func (f *fakeStore) List(_ context.Context, _ string, prefix string) ([]minio.ObjectInfo, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]minio.ObjectInfo, 0)
	for _, o := range f.objs {
		if strings.HasPrefix(o.Key, prefix) {
			out = append(out, o)
		}
	}
	return out, nil
}

func (f *fakeStore) Get(_ context.Context, _ string, key string) (io.ReadCloser, error) {
	b, ok := f.blobs[key]
	if !ok {
		return nil, errors.New("NoSuchKey")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func quietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func validSha() string { return strings.Repeat("ab", 32) } // 64 hex chars

func TestListerEmptyBucketReturnsEmptyCollection(t *testing.T) {
	li := Lister{MC: &fakeStore{}, Bucket: "canonical", L: quietLogger()}
	items, err := li.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if items == nil {
		t.Fatalf("List returned nil; want empty non-nil slice (marshals to \"data\": [])")
	}
	if len(items) != 0 {
		t.Fatalf("len = %d, want 0", len(items))
	}
}

func TestListerParsesSortsAndFillsAttributes(t *testing.T) {
	t1 := time.Date(2026, 7, 4, 12, 34, 56, 0, time.UTC)
	t2 := time.Date(2026, 7, 3, 1, 2, 3, 0, time.UTC)
	fs := &fakeStore{
		objs: []minio.ObjectInfo{
			// Deliberately out of order: JMS first, GMS 84 before GMS 83.
			{Key: DumpKey("JMS", 185, 1), Size: 300, LastModified: t2},
			{Key: DumpKey("GMS", 84, 1), Size: 200, LastModified: t2},
			{Key: DumpKey("GMS", 83, 1), Size: 100, LastModified: t1},
		},
		blobs: map[string][]byte{
			ShaKey("GMS", 83, 1): []byte(validSha()),
			ShaKey("GMS", 84, 1): []byte(validSha()),
			// JMS sidecar intentionally absent.
		},
	}
	li := Lister{MC: fs, Bucket: "canonical", L: quietLogger()}
	items, err := li.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d, want 3", len(items))
	}
	// Sorted by (region, major, minor) ascending.
	if items[0].Id != "GMS/83.1" || items[1].Id != "GMS/84.1" || items[2].Id != "JMS/185.1" {
		t.Fatalf("order = %s, %s, %s", items[0].Id, items[1].Id, items[2].Id)
	}
	first := items[0]
	if first.Region != "GMS" || first.MajorVersion != 83 || first.MinorVersion != 1 {
		t.Fatalf("first identity = %s/%d.%d", first.Region, first.MajorVersion, first.MinorVersion)
	}
	if first.Sha256 != validSha() {
		t.Fatalf("first sha = %q", first.Sha256)
	}
	if first.PublishedAt != "2026-07-04T12:34:56Z" {
		t.Fatalf("first publishedAt = %q", first.PublishedAt)
	}
	if first.SizeBytes != 100 {
		t.Fatalf("first size = %d", first.SizeBytes)
	}
	// Missing sidecar -> listed with empty sha, not dropped.
	if items[2].Sha256 != "" {
		t.Fatalf("JMS sha = %q, want empty", items[2].Sha256)
	}
}

func TestListerSkipsNonDumpAndUnparseableKeys(t *testing.T) {
	t1 := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	fs := &fakeStore{
		objs: []minio.ObjectInfo{
			{Key: DumpKey("GMS", 83, 1), Size: 100, LastModified: t1},
			{Key: ShaKey("GMS", 83, 1), Size: 64, LastModified: t1},                              // sidecar itself: not a dump
			{Key: "baseline/regions/GMS/notes.txt", Size: 5, LastModified: t1},                   // junk under prefix
			{Key: "baseline/regions/BAD/versions/x.y/documents.dump", Size: 5, LastModified: t1}, // unparseable version
		},
		blobs: map[string][]byte{ShaKey("GMS", 83, 1): []byte(validSha())},
	}
	li := Lister{MC: fs, Bucket: "canonical", L: quietLogger()}
	items, err := li.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].Id != "GMS/83.1" {
		t.Fatalf("items = %+v, want single GMS/83.1", items)
	}
}

func TestListerMalformedSidecarYieldsEmptySha(t *testing.T) {
	t1 := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	for name, blob := range map[string][]byte{
		"short":  []byte("abc123"),
		"nonhex": []byte(strings.Repeat("zz", 32)),
	} {
		fs := &fakeStore{
			objs:  []minio.ObjectInfo{{Key: DumpKey("GMS", 83, 1), Size: 100, LastModified: t1}},
			blobs: map[string][]byte{ShaKey("GMS", 83, 1): blob},
		}
		li := Lister{MC: fs, Bucket: "canonical", L: quietLogger()}
		items, err := li.List(context.Background())
		if err != nil {
			t.Fatalf("%s: List: %v", name, err)
		}
		if len(items) != 1 || items[0].Sha256 != "" {
			t.Fatalf("%s: items = %+v, want single entry with empty sha", name, items)
		}
	}
}

func TestListerSurfacesListError(t *testing.T) {
	li := Lister{MC: &fakeStore{listErr: errors.New("boom")}, Bucket: "canonical", L: quietLogger()}
	if _, err := li.List(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}
