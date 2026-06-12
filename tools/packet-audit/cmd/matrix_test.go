package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatrixSubcommandWritesOutputs(t *testing.T) {
	root, args := matrixTestRoot(t)

	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("matrix exit = %d", code)
	}
	md1, err := os.ReadFile(filepath.Join(root, "audits", "STATUS.md"))
	if err != nil {
		t.Fatalf("STATUS.md not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "audits", "status.json")); err != nil {
		t.Fatalf("status.json not written: %v", err)
	}
	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("second run exit = %d", code)
	}
	md2, _ := os.ReadFile(filepath.Join(root, "audits", "STATUS.md"))
	if !bytes.Equal(md1, md2) {
		t.Error("matrix output not deterministic")
	}
}

// matrixTestRoot sets up a standard single-version (gms_v83) temp tree and
// returns the root path and args slice (without --check).
func matrixTestRoot(t *testing.T) (root string, args []string) {
	t.Helper()
	root = t.TempDir()
	mustCopy(t, filepath.Join("..", "internal", "opregistry", "testdata", "good_version.yaml"),
		filepath.Join(root, "registry", "gms_v83.yaml"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "audits", "gms_v83", "Invite.json"),
		filepath.Join(root, "audits", "gms_v83", "Invite.json"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "templates", "template_gms_83_1.json"),
		filepath.Join(root, "templates", "template_gms_83_1.json"))
	mustCopy(t, filepath.Join("testdata", "gms_v95_mini.json"),
		filepath.Join(root, "exports", "gms_v83.json"))
	args = []string{
		"--registry-dir", filepath.Join(root, "registry"),
		"--audits-dir", filepath.Join(root, "audits"),
		"--templates-dir", filepath.Join(root, "templates"),
		"--exports-dir", filepath.Join(root, "exports"),
		"--versions", "gms_v83",
		"--out-dir", filepath.Join(root, "audits"),
	}
	return root, args
}

// TestMatrixCheckFreshPass: generate outputs, then --check should exit 0.
func TestMatrixCheckFreshPass(t *testing.T) {
	_, args := matrixTestRoot(t)

	// Generate.
	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("generate exit = %d", code)
	}
	// Check: should be fresh.
	checkArgs := append(args, "--check")
	if code := runMatrix(checkArgs, os.Stderr); code != 0 {
		t.Fatalf("--check exit = %d (want 0)", code)
	}
}

// TestMatrixCheckStaleFailure: mutate STATUS.md after generation, then --check must exit 1.
func TestMatrixCheckStaleFailure(t *testing.T) {
	root, args := matrixTestRoot(t)

	// Generate.
	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("generate exit = %d", code)
	}
	// Mutate STATUS.md to make it stale.
	mdPath := filepath.Join(root, "audits", "STATUS.md")
	if err := os.WriteFile(mdPath, []byte("stale content\n"), 0o644); err != nil {
		t.Fatalf("mutate STATUS.md: %v", err)
	}
	// Check: should detect staleness and return exitBlocker (1).
	var stderrBuf strings.Builder
	checkArgs := append(args, "--check")
	code := runMatrix(checkArgs, &stderrBuf)
	if code != exitBlocker {
		t.Fatalf("--check exit = %d (want %d); stderr: %s", code, exitBlocker, stderrBuf.String())
	}
	if !strings.Contains(stderrBuf.String(), "stale") {
		t.Errorf("expected 'stale' in stderr, got: %s", stderrBuf.String())
	}
}

// TestMatrixMissingTemplateWarning: a missing template must NOT fail the run,
// and must emit a warning to stderr.
func TestMatrixMissingTemplateWarning(t *testing.T) {
	root := t.TempDir()
	mustCopy(t, filepath.Join("..", "internal", "opregistry", "testdata", "good_version.yaml"),
		filepath.Join(root, "registry", "gms_v83.yaml"))
	// No template file provided — templates dir exists but is empty.
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}

	var stderrBuf strings.Builder
	args := []string{
		"--registry-dir", filepath.Join(root, "registry"),
		"--audits-dir", filepath.Join(root, "audits"),
		"--templates-dir", filepath.Join(root, "templates"),
		"--exports-dir", filepath.Join(root, "exports"),
		"--versions", "gms_v83",
		"--out-dir", filepath.Join(root, "audits"),
	}
	code := runMatrix(args, &stderrBuf)
	if code != 0 {
		t.Fatalf("matrix with missing template should exit 0, got %d; stderr: %s", code, stderrBuf.String())
	}
	if !strings.Contains(stderrBuf.String(), "warning") {
		t.Errorf("expected warning in stderr for missing template, got: %s", stderrBuf.String())
	}
}

// TestMatrixCheckConflictFails: a matrix with a conflict cell must make
// --check exit 1 mentioning "conflict" (design §10.1).
//
// Conflict condition (template-wiring gap, design §10.1):
//   - Two versions: gms_v83 and gms_v84.
//   - LOGIN_STATUS is Present in both registries: v83 at opcode 0x000, v84 at opcode 0x001.
//   - v83 template routes 0x000 (LOGIN_STATUS in v83) → routedVersions contains v83.
//   - v84 template does NOT route 0x001 → LOGIN_STATUS is unrouted in v84.
//   - v84 has an audit report for LOGIN_STATUS (resolved address) → Atlas implements it.
//   - Result: routedElsewhere=true (v83 routes it), hasReport=true, !routed(v84)
//     → template-wiring-gap conflict fires for v84.
func TestMatrixCheckConflictFails(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Registry: v83 has LOGIN_STATUS at opcode 0x000 clientbound.
	mustCopy(t, filepath.Join("..", "internal", "opregistry", "testdata", "good_version.yaml"),
		filepath.Join(root, "registry", "gms_v83.yaml"))
	// Registry: v84 has LOGIN_STATUS at opcode 0x001 clientbound (different opcode).
	v84Registry := "- op: LOGIN_STATUS\n  direction: clientbound\n  opcode: 0x001\n  fname: \"CLogin::OnCheckPasswordResult\"\n  provenance: csv-import\n"
	if err := os.WriteFile(filepath.Join(root, "registry", "gms_v84.yaml"), []byte(v84Registry), 0o644); err != nil {
		t.Fatal(err)
	}

	// v83 template routes opcode 0x000 clientbound (LOGIN_STATUS in v83).
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "templates", "template_gms_83_1.json"),
		filepath.Join(root, "templates", "template_gms_83_1.json"))
	// v84 template does NOT route 0x001 — LOGIN_STATUS is unrouted in v84.
	// This creates a template-wiring gap (routedElsewhere=true, hasReport=true).
	v84Template := `{"region":"GMS","majorVersion":84,"minorVersion":1,"socket":{"handlers":[],"writers":[]}}`
	if err := os.WriteFile(filepath.Join(root, "templates", "template_gms_84_1.json"), []byte(v84Template), 0o644); err != nil {
		t.Fatal(err)
	}

	// v84 audit report for LOGIN_STATUS: Atlas implements it (resolved address).
	// Writer "AuthResult" maps to FName "CLogin::OnCheckPasswordResult".
	if err := os.MkdirAll(filepath.Join(root, "audits", "gms_v84"), 0o755); err != nil {
		t.Fatal(err)
	}
	v84Report := `{"WriterName":"AuthResult","IDAName":"CLogin::OnCheckPasswordResult","Address":"0x5e9900","Variant":"GMS/v84","BranchDepth":0,"AtlasFile":"libs/atlas-packet/login/clientbound/auth_result.go","Rows":[],"Verdict":0,"FlatInvalid":false}`
	if err := os.WriteFile(filepath.Join(root, "audits", "gms_v84", "AuthResult.json"), []byte(v84Report), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--registry-dir", filepath.Join(root, "registry"),
		"--audits-dir", filepath.Join(root, "audits"),
		"--templates-dir", filepath.Join(root, "templates"),
		"--exports-dir", filepath.Join(root, "exports"),
		"--versions", "gms_v83,gms_v84",
		"--out-dir", filepath.Join(root, "audits"),
	}

	// Generate outputs first.
	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("generate exit = %d", code)
	}

	// --check must fail because of the conflict cell.
	var stderrBuf strings.Builder
	checkArgs := append(args, "--check")
	code := runMatrix(checkArgs, &stderrBuf)
	if code != exitBlocker {
		t.Fatalf("--check with conflict: exit = %d (want %d); stderr: %s", code, exitBlocker, stderrBuf.String())
	}
	if !strings.Contains(stderrBuf.String(), "conflict") {
		t.Errorf("expected 'conflict' in stderr, got: %s", stderrBuf.String())
	}
}

func mustCopy(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
