package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func buildLint(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "catalog-lint")
	out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return exe
}

func TestLint_GoodTreeExitsZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/good")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("expected exit 0, got %v", err)
	}
}

func TestLint_IDMismatchExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/id-mismatch")
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

func TestLint_MissingRevisionExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/missing-revision")
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}
