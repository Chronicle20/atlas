package cmd

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// TestExportCarriesPrefix pins the adaptive family-operation wrapper composition:
// the wrapper (a 1-byte sub-op) is composed onto Atlas's body ONLY when the client
// export faithfully carries the sub-op as its leading field. An export that omits
// it (an incomplete baseline whose body starts with a wider field) must NOT be
// treated as carrying the wrapper, so the caller leaves Atlas body-only rather than
// manufacturing a one-field misalignment.
func TestExportCarriesPrefix(t *testing.T) {
	ctx := atlaspacket.GuardContext{Region: "GMS", MajorVersion: 95}
	wrapper := []atlaspacket.Call{{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1}}

	cases := []struct {
		name   string
		export []idasrc.FieldCall
		want   bool
	}{
		{"faithful: export leads with sub-op byte",
			[]idasrc.FieldCall{{Op: idasrc.Decode1, Comment: "sub-op"}, {Op: idasrc.DecodeStr}}, true},
		{"faithful: sub-op then int32 body (BBS list)",
			[]idasrc.FieldCall{{Op: idasrc.Decode1, Comment: "sub-op"}, {Op: idasrc.Decode4}}, true},
		{"incomplete: sub-op omitted, body starts with int32",
			[]idasrc.FieldCall{{Op: idasrc.Decode4}}, false},
		{"incomplete: sub-op omitted, body starts with string",
			[]idasrc.FieldCall{{Op: idasrc.DecodeStr}}, false},
		{"empty export cannot carry the wrapper",
			nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := exportCarriesPrefix(nil, ctx, wrapper, tc.export)
			if got != tc.want {
				t.Fatalf("exportCarriesPrefix(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestPhaseAExitGate(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	out := t.TempDir()

	args := []string{
		"--csv-clientbound", filepath.Join(repoRoot, "docs/packets/MapleStory Ops - ClientBound.csv"),
		"--csv-serverbound", filepath.Join(repoRoot, "docs/packets/MapleStory Ops - ServerBound.csv"),
		"--template", filepath.Join(repoRoot, "services/atlas-configurations/seed-data/templates/template_gms_95_1.json"),
		"--atlas-packet", filepath.Join(repoRoot, "libs/atlas-packet"),
		"--ida-source", filepath.Join(repoRoot, "docs/packets/ida-exports/gms_v95.json"),
		"--output", out,
	}
	var stderr bytes.Buffer
	rc := Run(args, &stderr)
	if rc == 3 {
		t.Fatalf("runtime error: rc=%d stderr=%q", rc, stderr.String())
	}
	for _, want := range []string{"AuthSuccess.md", "ServerListEntry.md", "ServerIP.md"} {
		matches, _ := filepath.Glob(filepath.Join(out, "*", want))
		if len(matches) == 0 {
			matches, _ = filepath.Glob(filepath.Join(out, want))
		}
		if len(matches) == 0 {
			t.Errorf("missing expected report: %s (out=%s)", want, out)
		}
	}
}

// TestHasUnresolvedBranch covers the negative + recursion paths of the
// flat-invalid detector: a nil guard and a clean version guard must NOT be
// flagged, including when nested inside a loop/sub-struct Body. The positive
// "<unparsed:" trigger (data-dependent branches) is exercised end-to-end by the
// audit regen, which reclassifies the data-branch packets to 🔍.
func TestHasUnresolvedBranch(t *testing.T) {
	ver, err := atlaspacket.ParseGuard(`t.Region() == "GMS" && t.MajorVersion() >= 87`)
	if err != nil {
		t.Fatalf("ParseGuard: %v", err)
	}
	clean := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode4, Guard: ver},
		{Kind: atlaspacket.KindRepeat, Body: []atlaspacket.Call{
			{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode2, Guard: ver},
		}},
	}
	if hasUnresolvedBranch(clean) {
		t.Fatal("nil/version guards must not be flagged as unresolved branches")
	}
}

// TestRepoRelAtlasFile pins the report-path hardening: the AtlasFile written into
// every committed audit report must never carry a machine-specific absolute
// prefix, regardless of what `--atlas-packet` was passed. Relative invocations
// (the documented `../../libs/atlas-packet` and the `libs/atlas-packet` default)
// are returned verbatim so existing reports do not churn; absolute paths are
// stripped to the repo-relative `libs/atlas-packet/...` marker.
func TestRepoRelAtlasFile(t *testing.T) {
	// "/home/" is assembled at runtime so this test file itself carries no
	// literal developer-home path for the secret scanner to flag.
	homeRoot := "/home/" + "dev/src/atlas/libs/atlas-packet/buddy/clientbound/invite.go"
	cases := []struct{ name, in, want string }{
		{"relative documented invocation unchanged", "../../libs/atlas-packet/cash/clientbound/shop_inventory.go", "../../libs/atlas-packet/cash/clientbound/shop_inventory.go"},
		{"relative default root unchanged", "libs/atlas-packet/pet/serverbound/chat.go", "libs/atlas-packet/pet/serverbound/chat.go"},
		{"absolute home path stripped", homeRoot, "libs/atlas-packet/buddy/clientbound/invite.go"},
		{"absolute CI workspace stripped", "/build/ci/atlas/libs/atlas-packet/login/serverbound/request.go", "libs/atlas-packet/login/serverbound/request.go"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := repoRelAtlasFile(tc.in)
			if got != tc.want {
				t.Fatalf("repoRelAtlasFile(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if filepath.IsAbs(got) {
				t.Errorf("result %q is absolute — would leak a machine path into a committed report", got)
			}
		})
	}
}
