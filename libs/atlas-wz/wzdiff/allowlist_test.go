package wzdiff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadAllowlistParsesAndNormalizes covers the "want entries back" half of
// LoadAllowlist's contract: a well-formed file parses into the expected
// AllowEntry values, and path normalization (trailing/repeated trailing
// slash, surrounding whitespace) still produces entries that match via
// Allowed.
func TestLoadAllowlistParsesAndNormalizes(t *testing.T) {
	cases := []struct {
		name         string
		content      string
		wantEntries  []AllowEntry
		checkAllowed bool
		allowedImage string
		allowedDelta Delta
	}{
		{
			name: "basic two entries",
			content: "# comment line\n" +
				"\n" +
				"2519002\t/imgdir:0\treference\tUOL resolved by HaRepacker\n" +
				"2519003\t/imgdir:1/uol:hit\tours\tliteral UOL is correct\n",
			wantEntries: []AllowEntry{
				{Image: "2519002", Path: "/imgdir:0", OnlyIn: "reference", Reason: "UOL resolved by HaRepacker"},
				{Image: "2519003", Path: "/imgdir:1/uol:hit", OnlyIn: "ours", Reason: "literal UOL is correct"},
			},
		},
		{
			// A trailing slash on Path used to pass validation but could
			// never match: Allowed's prefix check becomes
			// d.Path == "/imgdir:0/" || strings.HasPrefix(d.Path, "/imgdir:0//"),
			// and no real delta path ever has a double slash.
			name:    "normalizes trailing slash on path",
			content: "2519002\t/imgdir:0/\treference\tsome reason\n",
			wantEntries: []AllowEntry{
				{Image: "2519002", Path: "/imgdir:0", OnlyIn: "reference", Reason: "some reason"},
			},
			checkAllowed: true,
			allowedImage: "2519002",
			allowedDelta: Delta{Path: "/imgdir:0", OnlyIn: "reference"},
		},
		{
			name:    "normalizes whitespace",
			content: " 2519002 \t /imgdir:0 \t reference \tsome reason\n",
			wantEntries: []AllowEntry{
				{Image: "2519002", Path: "/imgdir:0", OnlyIn: "reference", Reason: "some reason"},
			},
			checkAllowed: true,
			allowedImage: "2519002",
			allowedDelta: Delta{Path: "/imgdir:0", OnlyIn: "reference"},
		},
		{
			// "/imgdir:0//" (two trailing slashes) must normalize the same
			// as "/imgdir:0/" (one) does, to "/imgdir:0". A single-slash-
			// strip would leave "/imgdir:0/", which passes validation but
			// can never match via Allowed's HasPrefix(d.Path, e.Path+"/")
			// check, since no real delta path has a double slash.
			name:    "normalizes repeated trailing slash on path",
			content: "2519002\t/imgdir:0//\treference\tsome reason\n",
			wantEntries: []AllowEntry{
				{Image: "2519002", Path: "/imgdir:0", OnlyIn: "reference", Reason: "some reason"},
			},
			checkAllowed: true,
			allowedImage: "2519002",
			allowedDelta: Delta{Path: "/imgdir:0", OnlyIn: "reference"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "allowlist.tsv")
			if err := os.WriteFile(path, []byte(c.content), 0o644); err != nil {
				t.Fatalf("write allowlist: %v", err)
			}

			entries, err := LoadAllowlist(path)
			if err != nil {
				t.Fatalf("LoadAllowlist: %v", err)
			}
			if len(entries) != len(c.wantEntries) {
				t.Fatalf("len(entries) = %d, want %d", len(entries), len(c.wantEntries))
			}
			for i, want := range c.wantEntries {
				if entries[i] != want {
					t.Errorf("entries[%d] = %+v, want %+v", i, entries[i], want)
				}
			}
			if c.checkAllowed {
				if !Allowed(entries, c.allowedImage, c.allowedDelta) {
					t.Error("Allowed = false, want true for the normalized entry")
				}
			}
		})
	}
}

// TestLoadAllowlistRejectsInvalid covers the "want an error" half of
// LoadAllowlist's contract: malformed lines and fields that fail validation
// must all be rejected.
func TestLoadAllowlistRejectsInvalid(t *testing.T) {
	cases := []struct {
		name          string
		content       string
		wantErrSubstr string
	}{
		{
			name: "malformed line",
			content: "2519002\t/imgdir:0\treference\tgood line\n" +
				"2519003\t/imgdir:1\tonly-three-fields\n",
			wantErrSubstr: ":2:",
		},
		{
			name:    "blank image",
			content: "\t/imgdir:0\treference\tsome reason\n",
		},
		{
			// A blank Path is the dangerous case: Allowed's prefix check
			// (d.Path == e.Path || strings.HasPrefix(d.Path, e.Path+"/"))
			// would otherwise match every delta path on the image, since
			// every path is a descendant of "" + "/".
			name:    "blank path",
			content: "2519002\t\treference\tsome reason\n",
		},
		{
			name:    "blank onlyIn",
			content: "2519002\t/imgdir:0\t\tsome reason\n",
		},
		{
			name:    "invalid onlyIn",
			content: "2519002\t/imgdir:0\tboth\tsome reason\n",
		},
		{
			// "/" normalizes to "" (TrimRight strips all trailing "/"), and
			// an empty Path must be rejected by the existing blank-path
			// check rather than silently allowlisting every delta on the
			// image.
			name:    "bare slash path",
			content: "2519002\t/\treference\tsome reason\n",
		},
		{
			// "//" must normalize to "" the same as "/" does: TrimRight
			// strips ALL trailing slashes, not just one. A single-slash-
			// strip (TrimSuffix) would leave "/" behind, which passes the
			// blank-path check and silently allowlists every delta on the
			// image -- the same fail-closed footgun the trailing-slash fix
			// was meant to close.
			name:    "repeated slash path",
			content: "2519002\t//\treference\tsome reason\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "allowlist.tsv")
			if err := os.WriteFile(path, []byte(c.content), 0o644); err != nil {
				t.Fatalf("write allowlist: %v", err)
			}

			_, err := LoadAllowlist(path)
			if err == nil {
				t.Fatalf("LoadAllowlist: want error for %q, got nil", c.name)
			}
			if c.wantErrSubstr != "" && !strings.Contains(err.Error(), c.wantErrSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), c.wantErrSubstr)
			}
		})
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
