package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const schemaPath = "../../services/atlas-reactor-actions/docs/reactor_script_schema.json"

func buildLint(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "reactor-seed-lint")
	out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return exe
}

func TestLint_GoodCorpusExitsZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/good", schemaPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got %v\n%s", err, out)
	}
}

func TestLint_LegacyKeysExitNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/legacy-keys", schemaPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}
	if !strings.Contains(string(out), "mesoRange") && !strings.Contains(string(out), "additionalProperties") {
		t.Fatalf("expected output to mention mesoRange or additionalProperties, got:\n%s", out)
	}
}

func TestLint_TypeMismatchExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/type-mismatch", schemaPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}
	if !strings.Contains(string(out), "type") {
		t.Fatalf("expected output to mention type, got:\n%s", out)
	}
}

func TestLint_IDMismatchExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/id-mismatch", schemaPath)
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

func TestLint_MissingRequiredExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/missing-required", schemaPath)
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

func TestLint_MissingDescriptionExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/no-description", schemaPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}
	if !strings.Contains(string(out), "description") {
		t.Fatalf("expected output to mention description, got:\n%s", out)
	}
}

func TestLint_DivergentCopiesExitNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/divergent-copies", schemaPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}
	if !strings.Contains(string(out), "2001") {
		t.Fatalf("expected output to mention 2001, got:\n%s", out)
	}
}

func TestLint_MissingCopyExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/missing-copy", schemaPath)
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

func TestLint_EmptyRootExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, t.TempDir(), schemaPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit for a root with no reactor-*.json files, got exit 0\n%s", out)
	}
	if !strings.Contains(string(out), "no reactor-*.json files found") {
		t.Fatalf("expected output to explain that no files were found, got:\n%s", out)
	}
}

func TestLint_NonexistentRootExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, filepath.Join(t.TempDir(), "does-not-exist"), schemaPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit for a nonexistent root, got exit 0\n%s", out)
	}
	if strings.Contains(string(out), "nil pointer") || strings.Contains(string(out), "panic") {
		t.Fatalf("expected a clean error, not a panic, got:\n%s", out)
	}
}
