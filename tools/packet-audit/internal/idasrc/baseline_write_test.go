package idasrc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDispatch_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "base.json")
	// Newline-formatted (as real baselines are); the surgical writer anchors on
	// the "calls" field line.
	const in = "{\n" +
		" \"binary\": \"x\",\n" +
		" \"md5\": \"y\",\n" +
		" \"generated_at\": \"z\",\n" +
		" \"functions\": {\n" +
		"  \"A::B#One\": {\n" +
		"   \"address\": \"0x1\",\n" +
		"   \"direction\": \"clientbound\",\n" +
		"   \"calls\": [\n" +
		"    {\n" +
		"     \"op\": \"Decode1\",\n" +
		"     \"comment\": \"c\"\n" +
		"    }\n" +
		"   ]\n" +
		"  }\n" +
		" }\n" +
		"}\n"
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}

	updates := map[string]DispatchUpdate{
		"A::B#One": {Dispatch: []Selector{{Discriminator: "mode", Case: 9}}, Note: "agent-confirmed @0x1 mode==9"},
	}
	if err := WriteDispatch(p, updates); err != nil {
		t.Fatal(err)
	}

	src, err := NewExportSource(p)
	if err != nil {
		t.Fatal(err)
	}
	var found *BaselineEntry
	for _, e := range src.Entries() {
		if e.FName == "A::B#One" {
			ee := e
			found = &ee
		}
	}
	if found == nil {
		t.Fatal("entry lost after write")
	}
	if len(found.Dispatch) != 1 || found.Dispatch[0].Case != 9 || found.Dispatch[0].Discriminator != "mode" {
		t.Fatalf("dispatch not persisted: %+v", found.Dispatch)
	}
	// Original calls preserved.
	if len(found.HandCalls) != 1 || found.HandCalls[0].Op != Decode1 {
		t.Fatalf("calls mutated: %+v", found.HandCalls)
	}
}

func TestWriteDispatch_UnknownFName(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "base.json")
	const in = `{"binary":"x","md5":"y","generated_at":"z","functions":{` +
		`"A::B#One":{"address":"0x1","direction":"clientbound","calls":[]}}}`
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	err := WriteDispatch(p, map[string]DispatchUpdate{"Nope::Missing": {Dispatch: []Selector{{Case: 1}}}})
	if err == nil {
		t.Fatal("expected error for unknown FName")
	}
}

func TestWriteDispatch_PreservesOrderAndIndent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "base.json")
	in := "{\n" +
		" \"binary\": \"x\",\n" +
		" \"md5\": \"y\",\n" +
		" \"generated_at\": \"z\",\n" +
		" \"functions\": {\n" +
		"  \"Zeta::OnB#One\": {\n" +
		"   \"address\": \"0x1\",\n" +
		"   \"direction\": \"clientbound\",\n" +
		"   \"calls\": [\n" +
		"    {\n" +
		"     \"op\": \"Decode1\",\n" +
		"     \"comment\": \"c\"\n" +
		"    }\n" +
		"   ]\n" +
		"  },\n" +
		"  \"Alpha::OnA\": {\n" +
		"   \"address\": \"0x2\",\n" +
		"   \"direction\": \"clientbound\",\n" +
		"   \"calls\": []\n" +
		"  }\n" +
		" }\n" +
		"}\n"
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteDispatch(p, map[string]DispatchUpdate{}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != in {
		t.Fatalf("no-op WriteDispatch changed bytes.\n--- want ---\n%s\n--- got ---\n%s", in, got)
	}
	if err := WriteDispatch(p, map[string]DispatchUpdate{"Zeta::OnB#One": {Dispatch: []Selector{{Discriminator: "mode", Case: 3}}}}); err != nil {
		t.Fatal(err)
	}
	s := func() string { x, _ := os.ReadFile(p); return string(x) }()
	if strings.Index(s, "Zeta::OnB#One") > strings.Index(s, "Alpha::OnA") {
		t.Fatalf("key order not preserved:\n%s", s)
	}
	if !strings.Contains(s, "\"dispatch\"") || !strings.Contains(s, "\"case\": 3") {
		t.Fatalf("dispatch not written:\n%s", s)
	}
}

func TestMatchPythonEscaping(t *testing.T) {
	// Non-ASCII -> \uXXXX (lowercase); HTML metachars stay literal.
	got := string(matchPythonEscaping([]byte("a — b && c < d")))
	wantEsc := fmt.Sprintf(`\u%04x`, '—')
	if !strings.Contains(got, wantEsc) {
		t.Fatalf("em-dash should escape to %s: %q", wantEsc, got)
	}
	if strings.ContainsRune(got, '—') {
		t.Fatalf("literal em-dash should be gone: %q", got)
	}
	if !strings.Contains(got, "&& c < d") {
		t.Fatalf("HTML metachars should stay literal: %q", got)
	}
}

func TestWriteDispatch_PreservesLegacyNoteKey(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "base.json")
	// Legacy singular "note" key must survive a round-trip (re-emitted as "notes").
	const in = `{"binary":"x","md5":"y","generated_at":"z","functions":{` +
		`"A::B":{"address":"0x1","direction":"clientbound","note":"keep me","calls":[]}}}`
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteDispatch(p, map[string]DispatchUpdate{}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if !strings.Contains(string(got), "keep me") {
		t.Fatalf("legacy note text dropped:\n%s", got)
	}
}

// B1 regression: two functions with BYTE-IDENTICAL value objects; updating the
// SECOND must not corrupt the first (positional, not first-occurrence, splice).
func TestWriteDispatch_IdenticalSiblings(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "base.json")
	body := "{\n" +
		"   \"address\": \"0x1\",\n" +
		"   \"direction\": \"clientbound\",\n" +
		"   \"calls\": [\n    {\n     \"op\": \"Decode1\",\n     \"comment\": \"c\"\n    }\n   ]\n  }"
	in := "{\n \"binary\": \"x\",\n \"md5\": \"y\",\n \"generated_at\": \"z\",\n \"functions\": {\n" +
		"  \"A::First\": " + body + ",\n" +
		"  \"A::Second\": " + body + "\n }\n}\n"
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	// Update only the SECOND (identical-body) entry.
	if err := WriteDispatch(p, map[string]DispatchUpdate{"A::Second": {Dispatch: []Selector{{Discriminator: "mode", Case: 5}}}}); err != nil {
		t.Fatal(err)
	}
	src, err := NewExportSource(p)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, e := range src.Entries() {
		got[e.FName] = len(e.Dispatch)
	}
	if got["A::Second"] != 1 {
		t.Fatalf("A::Second should have the selector, got %d", got["A::Second"])
	}
	if got["A::First"] != 0 {
		t.Fatalf("A::First must be UNCHANGED, but got %d dispatch (B1 corruption)", got["A::First"])
	}
}

func TestPrependCall_SurgicalLeadingByte(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "base.json")
	in := "{\n \"binary\": \"x\",\n \"md5\": \"y\",\n \"generated_at\": \"z\",\n \"functions\": {\n" +
		"  \"A::Send\": {\n" +
		"   \"address\": \"0x1\",\n" +
		"   \"direction\": \"serverbound\",\n" +
		"   \"calls\": [\n" +
		"    {\n     \"op\": \"EncodeStr\",\n     \"comment\": \"msg\"\n    }\n" +
		"   ]\n  },\n" +
		"  \"B::Other\": {\n" +
		"   \"address\": \"0x2\",\n" +
		"   \"direction\": \"serverbound\",\n" +
		"   \"calls\": [\n    {\n     \"op\": \"Encode4\",\n     \"comment\": \"id\"\n    }\n   ]\n  }\n" +
		" }\n}\n"
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := PrependCall(p, map[string]CallSpec{
		"A::Send": {Op: "Encode1", Comment: "leading byte"},
	}); err != nil {
		t.Fatal(err)
	}
	src, err := NewExportSource(p)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]Primitive{}
	for _, e := range src.Entries() {
		ops := make([]Primitive, len(e.HandCalls))
		for i, c := range e.HandCalls {
			ops[i] = c.Op
		}
		got[e.FName] = ops
	}
	if len(got["A::Send"]) != 2 || got["A::Send"][0] != Decode1 || got["A::Send"][1] != DecodeStr {
		t.Fatalf("A::Send = %v, want [Decode1 DecodeStr]", got["A::Send"])
	}
	if len(got["B::Other"]) != 1 || got["B::Other"][0] != Decode4 {
		t.Fatalf("B::Other must be unchanged, got %v", got["B::Other"])
	}
}

// A compact "calls": [{ layout (first element not at line start) must ERROR, not
// silently emit invalid JSON.
func TestPrependCall_RejectsCompactCallsLayout(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "base.json")
	// Semi-compact: the first element's `{` sits on the `"calls": [` line, but the
	// fields are newline-formatted — the case that would mis-capture elemIndent.
	const in = "{\n \"binary\": \"x\",\n \"md5\": \"y\",\n \"generated_at\": \"z\",\n \"functions\": {\n" +
		"  \"A::Send\": {\n" +
		"   \"address\": \"0x1\",\n" +
		"   \"direction\": \"serverbound\",\n" +
		"   \"calls\": [{\n     \"op\": \"EncodeStr\",\n     \"comment\": \"m\"\n    }]\n  }\n }\n}\n"
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	err := PrependCall(p, map[string]CallSpec{"A::Send": {Op: "Encode1", Comment: "x"}})
	if err == nil {
		t.Fatal("expected error for compact calls layout, got nil (silent corruption risk)")
	}
}
