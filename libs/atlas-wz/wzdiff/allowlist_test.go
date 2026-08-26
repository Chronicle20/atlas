package wzdiff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAllowlist(t *testing.T) {
	content := "# comment line\n" +
		"\n" +
		"2519002\t/imgdir:0\treference\tUOL resolved by HaRepacker\n" +
		"2519003\t/imgdir:1/uol:hit\tours\tliteral UOL is correct\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	entries, err := LoadAllowlist(path)
	if err != nil {
		t.Fatalf("LoadAllowlist: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	want := AllowEntry{Image: "2519002", Path: "/imgdir:0", OnlyIn: "reference", Reason: "UOL resolved by HaRepacker"}
	if entries[0] != want {
		t.Errorf("entries[0] = %+v, want %+v", entries[0], want)
	}
	want2 := AllowEntry{Image: "2519003", Path: "/imgdir:1/uol:hit", OnlyIn: "ours", Reason: "literal UOL is correct"}
	if entries[1] != want2 {
		t.Errorf("entries[1] = %+v, want %+v", entries[1], want2)
	}
}

func TestLoadAllowlistMalformedLine(t *testing.T) {
	content := "2519002\t/imgdir:0\treference\tgood line\n" +
		"2519003\t/imgdir:1\tonly-three-fields\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	_, err := LoadAllowlist(path)
	if err == nil {
		t.Fatal("LoadAllowlist: want error for malformed line, got nil")
	}
	if !strings.Contains(err.Error(), ":2:") {
		t.Errorf("error %q does not name line 2", err.Error())
	}
}

func TestLoadAllowlistRejectsBlankImage(t *testing.T) {
	content := "\t/imgdir:0\treference\tsome reason\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	_, err := LoadAllowlist(path)
	if err == nil {
		t.Fatal("LoadAllowlist: want error for blank image field, got nil")
	}
}

func TestLoadAllowlistRejectsBlankPath(t *testing.T) {
	// A blank Path is the dangerous case: Allowed's prefix check
	// (d.Path == e.Path || strings.HasPrefix(d.Path, e.Path+"/")) would
	// otherwise match every delta path on the image, since every path is a
	// descendant of "" + "/".
	content := "2519002\t\treference\tsome reason\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	_, err := LoadAllowlist(path)
	if err == nil {
		t.Fatal("LoadAllowlist: want error for blank path field, got nil")
	}
}

func TestLoadAllowlistRejectsBlankOnlyIn(t *testing.T) {
	content := "2519002\t/imgdir:0\t\tsome reason\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	_, err := LoadAllowlist(path)
	if err == nil {
		t.Fatal("LoadAllowlist: want error for blank onlyIn field, got nil")
	}
}

func TestLoadAllowlistRejectsInvalidOnlyIn(t *testing.T) {
	content := "2519002\t/imgdir:0\tboth\tsome reason\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	_, err := LoadAllowlist(path)
	if err == nil {
		t.Fatal("LoadAllowlist: want error for invalid onlyIn field, got nil")
	}
}

func TestLoadAllowlistNormalizesTrailingSlashOnPath(t *testing.T) {
	// A trailing slash on Path used to pass validation but could never
	// match: Allowed's prefix check becomes d.Path == "/imgdir:0/" ||
	// strings.HasPrefix(d.Path, "/imgdir:0//"), and no real delta path ever
	// has a double slash.
	content := "2519002\t/imgdir:0/\treference\tsome reason\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	entries, err := LoadAllowlist(path)
	if err != nil {
		t.Fatalf("LoadAllowlist: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Path != "/imgdir:0" {
		t.Errorf("entries[0].Path = %q, want %q", entries[0].Path, "/imgdir:0")
	}
	if !Allowed(entries, "2519002", Delta{Path: "/imgdir:0", OnlyIn: "reference"}) {
		t.Error("Allowed = false, want true for the normalized path")
	}
}

func TestLoadAllowlistNormalizesWhitespace(t *testing.T) {
	content := " 2519002 \t /imgdir:0 \t reference \tsome reason\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	entries, err := LoadAllowlist(path)
	if err != nil {
		t.Fatalf("LoadAllowlist: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	want := AllowEntry{Image: "2519002", Path: "/imgdir:0", OnlyIn: "reference", Reason: "some reason"}
	if entries[0] != want {
		t.Errorf("entries[0] = %+v, want %+v", entries[0], want)
	}
	// The normalization-then-Allowed round trip: a whitespace-padded entry
	// now matches the delta it was written for.
	if !Allowed(entries, "2519002", Delta{Path: "/imgdir:0", OnlyIn: "reference"}) {
		t.Error("Allowed = false, want true for the normalized, whitespace-padded entry")
	}
}

func TestLoadAllowlistRejectsBareSlashPath(t *testing.T) {
	// "/" normalizes to "" (TrimRight strips all trailing "/"), and an empty
	// Path must be rejected by the existing blank-path check rather than
	// silently allowlisting every delta on the image.
	content := "2519002\t/\treference\tsome reason\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	_, err := LoadAllowlist(path)
	if err == nil {
		t.Fatal("LoadAllowlist: want error for path \"/\", got nil")
	}
}

func TestLoadAllowlistRejectsRepeatedSlashPath(t *testing.T) {
	// "//" must normalize to "" the same as "/" does: TrimRight strips ALL
	// trailing slashes, not just one. A single-slash-strip (TrimSuffix)
	// would leave "/" behind, which passes the blank-path check and
	// silently allowlists every delta on the image -- the same
	// fail-closed footgun the trailing-slash fix was meant to close.
	content := "2519002\t//\treference\tsome reason\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	_, err := LoadAllowlist(path)
	if err == nil {
		t.Fatal("LoadAllowlist: want error for path \"//\", got nil")
	}
}

func TestLoadAllowlistNormalizesRepeatedTrailingSlashOnPath(t *testing.T) {
	// "/imgdir:0//" (two trailing slashes) must normalize the same as
	// "/imgdir:0/" (one) does, to "/imgdir:0". A single-slash-strip would
	// leave "/imgdir:0/", which passes validation but can never match via
	// Allowed's HasPrefix(d.Path, e.Path+"/") check, since no real delta
	// path has a double slash.
	content := "2519002\t/imgdir:0//\treference\tsome reason\n"
	path := filepath.Join(t.TempDir(), "allowlist.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	entries, err := LoadAllowlist(path)
	if err != nil {
		t.Fatalf("LoadAllowlist: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Path != "/imgdir:0" {
		t.Errorf("entries[0].Path = %q, want %q", entries[0].Path, "/imgdir:0")
	}
	if !Allowed(entries, "2519002", Delta{Path: "/imgdir:0", OnlyIn: "reference"}) {
		t.Error("Allowed = false, want true for the normalized path")
	}
}

func TestAllowed(t *testing.T) {
	entry := AllowEntry{Image: "2519002", Path: "/imgdir:0", OnlyIn: "reference", Reason: "UOL resolved by HaRepacker"}
	entries := []AllowEntry{entry}

	cases := []struct {
		name  string
		image string
		delta Delta
		want  bool
	}{
		{"exact match", "2519002", Delta{Path: "/imgdir:0", OnlyIn: "reference"}, true},
		{"wrong direction", "2519002", Delta{Path: "/imgdir:0", OnlyIn: "ours"}, false},
		{"wrong image", "2519003", Delta{Path: "/imgdir:0", OnlyIn: "reference"}, false},
		{"prefix, not exact", "2519002", Delta{Path: "/imgdir:0/canvas:0", OnlyIn: "reference"}, true},
		{"unrelated", "2519002", Delta{Path: "/imgdir:1/int:state", OnlyIn: "reference"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Allowed(entries, c.image, c.delta)
			if got != c.want {
				t.Errorf("Allowed(%q, %+v) = %v, want %v", c.image, c.delta, got, c.want)
			}
		})
	}
}
