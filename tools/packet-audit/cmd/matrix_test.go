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
