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
