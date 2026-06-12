package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMatrixSubcommandWritesOutputs(t *testing.T) {
	root := t.TempDir()
	// Layout: registry/, audits/gms_v83/, templates dir, exports dir.
	mustCopy(t, filepath.Join("..", "internal", "opregistry", "testdata", "good_version.yaml"),
		filepath.Join(root, "registry", "gms_v83.yaml"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "audits", "gms_v83", "Invite.json"),
		filepath.Join(root, "audits", "gms_v83", "Invite.json"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "templates", "template_gms_83_1.json"),
		filepath.Join(root, "templates", "template_gms_83_1.json"))
	mustCopy(t, filepath.Join("testdata", "gms_v95_mini.json"),
		filepath.Join(root, "exports", "gms_v83.json"))

	args := []string{
		"--registry-dir", filepath.Join(root, "registry"),
		"--audits-dir", filepath.Join(root, "audits"),
		"--templates-dir", filepath.Join(root, "templates"),
		"--exports-dir", filepath.Join(root, "exports"),
		"--versions", "gms_v83",
		"--out-dir", filepath.Join(root, "audits"),
	}
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
