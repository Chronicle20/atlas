package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// A seeded raw boundary comparison (`MajorVersion() > 83`) is flagged; the
// MajorAtLeast form is clean; a non-boundary constant (`> 12`) and an
// allowlisted line are not flagged. (task-169 T4.1 / FR-3.1a — both directions.)
func TestGateLintFlagsBoundaryComparisons(t *testing.T) {
	dir := t.TempDir()
	src := `package p

func f(t T) {
	if t.MajorVersion() > 83 {          // FLAG: boundary off-by-one footgun
	}
	if t.MajorAtLeast(87) {             // clean: uses the helper
	}
	if t.MajorVersion() > 12 {          // clean: 12 is a base-version gate, not a boundary
	}
	if t.MajorVersion() >= 95 {         //gate-lint:allow verified byte-shift
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "codec.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	hits, err := collectGateLintHits(gateLintConfig{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("want exactly 1 hit (the `> 83`), got %d: %+v", len(hits), hits)
	}
	if hits[0].boundary != 83 {
		t.Errorf("hit boundary = %d, want 83", hits[0].boundary)
	}
	if hits[0].line != 4 {
		t.Errorf("hit line = %d, want 4", hits[0].line)
	}
}

// The number-on-the-left form and <= / >= operators are also flagged when the
// constant is a real boundary; _test.go files are skipped.
func TestGateLintFormsAndTestSkip(t *testing.T) {
	dir := t.TempDir()
	src := `package p

func g(t T) {
	if 87 <= t.MajorVersion() {         // FLAG: number-on-left, boundary 87
	}
	if t.MajorVersion() < 79 {          // FLAG: boundary 79
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "wire.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	// A _test.go file with a boundary comparison must be ignored.
	if err := os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte("package p\n// t.MajorVersion() > 83\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hits, err := collectGateLintHits(gateLintConfig{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("want 2 hits (87-left, <79), got %d: %+v", len(hits), hits)
	}
}
