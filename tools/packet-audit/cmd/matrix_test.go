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
// Conflict condition: registry says LOGIN_STATUS is Present in gms_v84 (own
// file with one entry at 0x000 clientbound), and the template routes opcode
// 0x000 clientbound. But we ADD a second version gms_v84 whose registry file
// is empty while the template routes 0x000 → that produces a routing-gap
// conflict ("op present in client and routed in another version's template,
// but unrouted here").
//
// Simpler approach: two versions (v83 + v84). v83 template routes 0x02 cb
// (AccountInfo). v84 registry file exists (Absent applicability for LOGIN_STATUS)
// but v84 template also routes 0x000 cb. Because routedAnywhere[{0x000, cb}]=true
// (v83's template routes it) but v84's template also routes 0x000 cb too — so
// no gap there.
//
// Cleanest: one version, registry says LOGIN_STATUS Present at opcode 0x000 cb.
// Template routes opcode 0x001 cb. For opcode 0x001, the registry has NO entry
// at 0x001 — but the grading is based on op-name applicability, not opcode.
//
// Actually the simplest: two versions. v83 template routes op X. v84 registry
// file exists but v84 template does NOT route op X → routing-gap conflict for v84.
func TestMatrixCheckConflictFails(t *testing.T) {
	root := t.TempDir()

	// Registry: LOGIN_STATUS present in both gms_v83 and gms_v84 at opcode 0x000 clientbound.
	mustCopy(t, filepath.Join("..", "internal", "opregistry", "testdata", "good_version.yaml"),
		filepath.Join(root, "registry", "gms_v83.yaml"))
	// gms_v84 registry: same op so it's Present in both.
	mustCopy(t, filepath.Join("..", "internal", "opregistry", "testdata", "good_version.yaml"),
		filepath.Join(root, "registry", "gms_v84.yaml"))

	// v83 template routes opcode 0x000 clientbound (LOGIN_STATUS): routedAnywhere = true.
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "templates", "template_gms_83_1.json"),
		filepath.Join(root, "templates", "template_gms_83_1.json"))
	// v84 template: does NOT route 0x000 clientbound — produces routing-gap conflict.
	// Use a template with only a handler so no clientbound ops are routed.
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	noWriterTemplate := `{"region":"GMS","majorVersion":84,"minorVersion":1,"socket":{"handlers":[{"opCode":"0x01","handler":"SomeHandle"}],"writers":[]}}`
	if err := os.WriteFile(filepath.Join(root, "templates", "template_gms_84_1.json"), []byte(noWriterTemplate), 0o644); err != nil {
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
