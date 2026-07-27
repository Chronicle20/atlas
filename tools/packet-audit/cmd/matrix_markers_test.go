package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/evidence"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
)

// appendLine opens dst in O_APPEND mode and writes s.
func appendLine(t *testing.T, dst, s string) {
	t.Helper()
	f, err := os.OpenFile(dst, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
}

// findOpCell scans m.Rows for Kind==RowOp && Op==op and returns r.Cells[version].
// It fails the test when the row or the cell is absent.
func findOpCell(t *testing.T, m matrix.Matrix, op, version string) matrix.Cell {
	t.Helper()
	for _, r := range m.Rows {
		if r.Kind == matrix.RowOp && r.Op == op {
			c, ok := r.Cells[version]
			if !ok {
				t.Fatalf("op %s has no cell for version %s", op, version)
			}
			return c
		}
	}
	t.Fatalf("op %s not found in matrix rows", op)
	return matrix.Cell{}
}

func TestMatrixMarkerPromotionAndOrphanCheck(t *testing.T) {
	root := t.TempDir()
	// Same tree as TestMatrixSubcommandWritesOutputs...
	mustCopy(t, filepath.Join("..", "internal", "opregistry", "testdata", "good_version.yaml"),
		filepath.Join(root, "registry", "gms_v83.yaml"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "audits", "gms_v83", "Invite.json"),
		filepath.Join(root, "audits", "gms_v83", "Invite.json"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "templates", "template_gms_83_1.json"),
		filepath.Join(root, "templates", "template_gms_83_1.json"))
	mustCopy(t, filepath.Join("testdata", "gms_v95_mini.json"),
		filepath.Join(root, "exports", "gms_v83.json"))
	// Extend the registry YAML with a BUDDY_RESULT entry whose FName matches Invite.json's IDAName base.
	appendLine(t, filepath.Join(root, "registry", "gms_v83.yaml"),
		"- op: BUDDY_RESULT\n  direction: clientbound\n  opcode: 0x03F\n  fname: \"CWvsContext::OnFriendResult\"\n  provenance: csv-import\n")
	// Extend the template with a writer for opcode 0x03F so routedAnywhere=true.
	appendLineJSON(t, filepath.Join(root, "templates", "template_gms_83_1.json"), 0x03F)

	// Create a packet lib with a marker matching Invite.json's Address.
	lib := filepath.Join(root, "packetlib", "buddy", "clientbound")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	markerContent := "package clientbound\n\n// packet-audit:verify packet=buddy/clientbound/Invite version=gms_v83 ida=0xa3f2e8\nfunc TestX(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(lib, "invite_test.go"), []byte(markerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--registry-dir", filepath.Join(root, "registry"),
		"--audits-dir", filepath.Join(root, "audits"),
		"--templates-dir", filepath.Join(root, "templates"),
		"--exports-dir", filepath.Join(root, "exports"),
		"--evidence-dir", filepath.Join(root, "evidence"), // empty
		"--packet-lib", filepath.Join(root, "packetlib"),
		"--versions", "gms_v83",
		"--out-dir", filepath.Join(root, "audits"),
	}
	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("matrix exit = %d", code)
	}
	var m matrix.Matrix
	raw, _ := os.ReadFile(filepath.Join(root, "audits", "status.json"))
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	cell := findOpCell(t, m, "BUDDY_RESULT", "gms_v83")
	if cell.State.Name() != "verified" {
		t.Errorf("Invite cell = %s (%s), want verified", cell.State.Name(), cell.Note)
	}

	// Orphan: address matches nothing -> --check exits 1.
	orphanContent := "package clientbound\n\n// packet-audit:verify packet=buddy/clientbound/Invite version=gms_v83 ida=0xdeadbeef\nfunc TestX(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(lib, "invite_test.go"), []byte(orphanContent), 0o644); err != nil {
		t.Fatal(err)
	}
	// Regenerate with orphan marker so outputs are fresh.
	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("matrix with orphan exit = %d (orphan must not fail plain generation)", code)
	}
	var errBuf bytes.Buffer
	if code := runMatrix(append(args, "--check"), &errBuf); code == 0 {
		t.Fatal("orphan marker must fail --check")
	}
	if !strings.Contains(errBuf.String(), "orphan marker") {
		t.Errorf("stderr = %q", errBuf.String())
	}
}

// TestMatrixDanglingEvidenceFailsCheck verifies that an evidence record with no
// corresponding audit report is reported as a --check problem.
func TestMatrixDanglingEvidenceFailsCheck(t *testing.T) {
	root, args := matrixTestRoot(t)

	// Write an evidence record for a packet that has no audit report.
	evDir := filepath.Join(root, "evidence", "gms_v83")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	evRec := "packet: login/clientbound/Nonexistent\ndirection: clientbound\nversion: gms_v83\ncategory: OPAQUE\nida:\n  function: \"CLogin::OnFoo\"\n  address: \"0x1\"\n  decompile_sha256: \"abc123\"\n"
	if err := os.WriteFile(filepath.Join(evDir, "login.clientbound.Nonexistent.yaml"), []byte(evRec), 0o644); err != nil {
		t.Fatal(err)
	}
	// Point evidence dir at the new location.
	evArgs := make([]string, len(args))
	copy(evArgs, args)
	evArgs = append(evArgs,
		"--evidence-dir", filepath.Join(root, "evidence"),
		"--exports-dir", filepath.Join(root, "exports"),
	)

	// Generation succeeds (dangling evidence is only a --check failure).
	if code := runMatrix(evArgs, os.Stderr); code != 0 {
		t.Fatalf("generate exit = %d", code)
	}

	var errBuf strings.Builder
	if code := runMatrix(append(evArgs, "--check"), &errBuf); code == 0 {
		t.Fatal("dangling evidence must fail --check")
	}
	if !strings.Contains(errBuf.String(), "dangling") {
		t.Errorf("stderr = %q", errBuf.String())
	}
}

// TestMatrixPacketLinkedEvidenceExemptFromDangling verifies that an evidence
// record for a packet declared via a registry op's `packet:` field is NOT
// flagged as dangling by --check even though it has no audit report — that is
// the no-report byte-fixture promotion path (commit 6c202cb7).
func TestMatrixPacketLinkedEvidenceExemptFromDangling(t *testing.T) {
	root, args := matrixTestRoot(t)

	// Declare an op that carries a `packet:` field but has no audit report.
	appendLine(t, filepath.Join(root, "registry", "gms_v83.yaml"),
		"- op: FOO_PACKET\n  direction: clientbound\n  opcode: 0x0AA\n  fname: \"CLogin::OnFoo\"\n  packet: login/clientbound/FooPacket\n  provenance: csv-import\n")

	// Fresh evidence for that packet (CLogin::OnFoo is present in the mini export).
	exp := filepath.Join(root, "exports", "gms_v83.json")
	hash, err := evidence.FunctionHash(exp, "CLogin::OnFoo")
	if err != nil {
		t.Fatal(err)
	}
	evDir := filepath.Join(root, "evidence", "gms_v83")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	evRec := "packet: login/clientbound/FooPacket\ndirection: clientbound\nversion: gms_v83\ncategory: TIER1-FIXTURE\nida:\n  function: \"CLogin::OnFoo\"\n  address: \"0x1\"\n  decompile_sha256: \"" + hash + "\"\n"
	if err := os.WriteFile(filepath.Join(evDir, "login.clientbound.FooPacket.yaml"), []byte(evRec), 0o644); err != nil {
		t.Fatal(err)
	}

	evArgs := append(append([]string{}, args...), "--evidence-dir", filepath.Join(root, "evidence"))
	if code := runMatrix(evArgs, os.Stderr); code != 0 {
		t.Fatalf("generate exit = %d", code)
	}
	var errBuf strings.Builder
	if code := runMatrix(append(evArgs, "--check"), &errBuf); code != 0 {
		t.Fatalf("--check exit = %d (packet-linked report-less evidence must be exempt); stderr=%q", code, errBuf.String())
	}
	if strings.Contains(errBuf.String(), "dangling") {
		t.Errorf("packet-linked evidence must not be flagged dangling; stderr=%q", errBuf.String())
	}
}

// appendLineJSON rewrites the template JSON at path to add an extra writer
// entry with opCode 0x03F so routedAnywhere=true in the test matrix run.
func appendLineJSON(t *testing.T, path string, _ int) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Inject a new writer entry before the closing ] of the writers array.
	// The fixture template has "AccountInfo"\n      }\n    ] at the end.
	const needle = `"AccountInfo"` + "\n      }\n    ]"
	const inject = `"AccountInfo"` + "\n      },\n      {\n        \"opCode\": \"0x03F\",\n        \"writer\": \"BuddyResult\"\n      }\n    ]"
	patched := strings.Replace(string(raw), needle, inject, 1)
	if patched == string(raw) {
		t.Fatal("appendLineJSON: failed to patch template JSON — pattern not found")
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
}
