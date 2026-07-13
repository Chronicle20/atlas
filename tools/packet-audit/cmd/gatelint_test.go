package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// The off-by-one-prone strict `MajorVersion() > 83` is flagged; the MajorAtLeast
// form is clean; a non-boundary constant (`> 12`) is not flagged; an allowlisted
// line is suppressed; and — post-T4.1b narrowing — the CORRECT `>= N` idiom is
// NOT flagged even without an allow annotation. (task-169 T4.1b / FR-3.1a — both
// directions.)
func TestGateLintFlagsBoundaryComparisons(t *testing.T) {
	dir := t.TempDir()
	src := `package p

func f(t T) {
	if t.MajorVersion() > 83 {          // FLAG: strict > at a boundary — the footgun
	}
	if t.MajorAtLeast(87) {             // clean: uses the helper
	}
	if t.MajorVersion() > 12 {          // clean: 12 is a base-version gate, not a boundary
	}
	if t.MajorVersion() >= 95 {         // clean (narrowed): >= is the correct idiom, not flagged
	}
	if t.MajorVersion() < 87 {          // clean (narrowed): < is the correct idiom, not flagged
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
	if hits[0].boundary != 83 || hits[0].op != ">" {
		t.Errorf("hit = %s %d, want > 83", hits[0].op, hits[0].boundary)
	}
	if hits[0].line != 4 {
		t.Errorf("hit line = %d, want 4", hits[0].line)
	}
}

// The narrowed detector flags the two footgun forms — right-form `<= N` and its
// left-operand twin `N >= MajorVersion()` — but NOT the correct idioms `>= N` /
// `< N` (right) or their twins `<= N` / `> N` (left). `_test.go` is skipped.
func TestGateLintFormsAndTestSkip(t *testing.T) {
	dir := t.TempDir()
	src := `package p

func g(t T) {
	if t.MajorVersion() <= 87 {         // FLAG: inclusive <= at a boundary — footgun
	}
	if 83 >= t.MajorVersion() {         // FLAG: left twin of <= 83 — footgun
	}
	if 87 <= t.MajorVersion() {         // clean: twin of >= 87 (correct idiom)
	}
	if t.MajorVersion() < 79 {          // clean: < is the correct idiom
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
		t.Fatalf("want 2 hits (<=87, 83>=), got %d: %+v", len(hits), hits)
	}
}
