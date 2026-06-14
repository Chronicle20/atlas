package matrix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/evidence"
)

// writeMini writes an export with one function entry whose body we can mutate
// to simulate drift.
func writeMini(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "gms_v83.json")
	content := `{"binary":"x","md5":"0","generated_at":"0","functions":{"CLogin::OnFoo":` + body + `}}`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEvidenceInputFreshAndDrifted(t *testing.T) {
	dir := t.TempDir()
	exp := writeMini(t, dir, `{"address":"0x1","direction":"clientbound","calls":[{"op":"Decode4","comment":"a"}]}`)

	evDir := filepath.Join(dir, "evidence")
	os.MkdirAll(filepath.Join(evDir, "gms_v83"), 0o755)
	// Pin against the current export via the shared hash helper.
	rec := "packet: login/clientbound/Foo\ndirection: clientbound\nversion: gms_v83\ncategory: OPAQUE\nida:\n  function: \"CLogin::OnFoo\"\n  address: \"0x1\"\n  decompile_sha256: \"HASH\"\n"
	h := mustHash(t, exp, "CLogin::OnFoo")
	os.WriteFile(filepath.Join(evDir, "gms_v83", "login.clientbound.Foo.yaml"),
		[]byte(strings.ReplaceAll(rec, "HASH", h)), 0o644)

	st, problems, err := BuildEvidenceInputs(evDir, map[string]string{"gms_v83": exp})
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if !st[EvKey{"login/clientbound/Foo", "gms_v83"}].Fresh {
		t.Error("expected fresh evidence")
	}

	// Mutate the export -> drift.
	writeMini(t, dir, `{"address":"0x1","direction":"clientbound","calls":[{"op":"Decode1","comment":"CHANGED"}]}`)
	st2, problems2, _ := BuildEvidenceInputs(evDir, map[string]string{"gms_v83": exp})
	if st2[EvKey{"login/clientbound/Foo", "gms_v83"}].Fresh {
		t.Error("expected stale evidence after export change")
	}
	if len(problems2) == 0 {
		t.Error("drift must be reported as a check problem")
	}
}

func mustHash(t *testing.T, exp, fn string) string {
	t.Helper()
	h, err := evidence.FunctionHash(exp, fn)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
