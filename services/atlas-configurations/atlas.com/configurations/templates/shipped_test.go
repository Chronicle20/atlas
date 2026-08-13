package templates

import (
	"path/filepath"
	"testing"
)

// seedTemplatesDir is the checked-in seed corpus, relative to this package
// directory (where `go test` runs).
func seedTemplatesDir() string {
	return filepath.Join("..", "..", "..", "seed-data", "templates")
}

// FR-1.5 / FR-1.6: a directory containing an unparseable file, a file missing
// the required region, and a non-.json file must still yield the one good
// entry. A bad file is logged and omitted; it never fails the load.
func TestLoadCatalogTolerantOfBadFiles(t *testing.T) {
	c := LoadCatalog(testLogger(), filepath.Join("testdata", "templates"))

	if c.Len() != 1 {
		t.Fatalf("Len() = %d, want 1 (entries: %v)", c.Len(), c.Entries())
	}
	e, ok := c.Lookup("TEST", 1, 0)
	if !ok {
		t.Fatalf("Lookup(TEST,1,0) = miss, want hit")
	}
	if e.FileName != "valid_template.json" {
		t.Errorf("FileName = %q, want valid_template.json", e.FileName)
	}
	if e.Model.Region != "TEST" {
		t.Errorf("Model.Region = %q, want TEST", e.Model.Region)
	}
	want, err := Revision(e.Model)
	if err != nil {
		t.Fatalf("Revision: %v", err)
	}
	if e.Revision != want {
		t.Errorf("Revision = %q, want %q", e.Revision, want)
	}
}

// FR-1.6: two files resolving to the same key - the first in file-name sort
// order wins. a_first.json sorts before b_second.json and carries
// usesPin:false, so the winner is identifiable.
func TestLoadCatalogDuplicateKeyFirstWins(t *testing.T) {
	c := LoadCatalog(testLogger(), filepath.Join("testdata", "duplicates"))

	if c.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", c.Len())
	}
	e, ok := c.Lookup("DUP", 7, 3)
	if !ok {
		t.Fatalf("Lookup(DUP,7,3) = miss, want hit")
	}
	if e.FileName != "a_first.json" {
		t.Errorf("FileName = %q, want a_first.json (sort-order first)", e.FileName)
	}
	if e.Model.UsesPin {
		t.Errorf("UsesPin = true, want false - b_second.json overwrote a_first.json")
	}
}

// A missing directory is not an error: an empty catalog reports "no shipped
// file" for every template (FR-2.4), which is the safe degradation.
func TestLoadCatalogMissingDirectoryIsEmpty(t *testing.T) {
	c := LoadCatalog(testLogger(), filepath.Join("testdata", "does-not-exist"))
	if c.Len() != 0 {
		t.Errorf("Len() = %d, want 0", c.Len())
	}
	if _, ok := c.Lookup("TEST", 1, 0); ok {
		t.Errorf("Lookup on empty catalog = hit, want miss")
	}
}

// The zero Catalog must be usable - ProcessorImpl without WithCatalog holds
// one, and it must degrade to "no shipped file" rather than panic.
func TestZeroCatalogIsUsable(t *testing.T) {
	var c Catalog
	if c.Len() != 0 {
		t.Errorf("Len() = %d, want 0", c.Len())
	}
	if _, ok := c.Lookup("GMS", 83, 1); ok {
		t.Errorf("Lookup on zero catalog = hit, want miss")
	}
	if got := len(c.Entries()); got != 0 {
		t.Errorf("len(Entries()) = %d, want 0", got)
	}
}

// The real corpus: all eleven shipped seed files load, and the two versions
// that have historically been confused (GMS 83.1 and GMS 84.1) resolve to
// distinct keys. Migrated from seeder_test.go's TestExtractMetadataGmsV84 /
// TestGmsV84DistinctFromV83 / TestSeedDataDiscoversBothV83AndV84, which tested
// the now-deleted extractMetadata/discoverFiles path.
func TestLoadCatalogRealSeedCorpus(t *testing.T) {
	c := LoadCatalog(testLogger(), seedTemplatesDir())

	if c.Len() != 11 {
		t.Fatalf("Len() = %d, want 11 - a seed file was added or removed without updating this test", c.Len())
	}

	e83, ok := c.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("Lookup(GMS,83,1) = miss")
	}
	if e83.FileName != "template_gms_83_1.json" {
		t.Errorf("GMS 83.1 FileName = %q, want template_gms_83_1.json", e83.FileName)
	}

	e84, ok := c.Lookup("GMS", 84, 1)
	if !ok {
		t.Fatalf("Lookup(GMS,84,1) = miss")
	}
	if e84.FileName != "template_gms_84_1.json" {
		t.Errorf("GMS 84.1 FileName = %q, want template_gms_84_1.json", e84.FileName)
	}

	if e83.Revision == e84.Revision {
		t.Errorf("GMS 83.1 and GMS 84.1 share a revision - they are not distinct documents")
	}

	// Entries() must be deterministic and complete.
	entries := c.Entries()
	if len(entries) != c.Len() {
		t.Fatalf("len(Entries()) = %d, Len() = %d", len(entries), c.Len())
	}
	for i := 1; i < len(entries); i++ {
		if entries[i-1].FileName > entries[i].FileName {
			t.Errorf("Entries() not sorted by file name: %q before %q", entries[i-1].FileName, entries[i].FileName)
		}
	}
}

// The singleton wrapper initializes once and thereafter serves the same
// catalog (FR-1.2). It is exercised here rather than mocked because main.go
// is its only other caller.
func TestInitShippedCatalogIsIdempotent(t *testing.T) {
	first := InitShippedCatalog(testLogger(), seedTemplatesDir())
	if first.Len() != 11 {
		t.Fatalf("first Len() = %d, want 11", first.Len())
	}
	// A second call with a bogus directory must NOT replace the loaded catalog.
	second := InitShippedCatalog(testLogger(), filepath.Join("testdata", "does-not-exist"))
	if second.Len() != first.Len() {
		t.Errorf("second InitShippedCatalog changed the catalog: Len() = %d, want %d", second.Len(), first.Len())
	}
	if ShippedCatalog().Len() != first.Len() {
		t.Errorf("ShippedCatalog().Len() = %d, want %d", ShippedCatalog().Len(), first.Len())
	}
}
