package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWrap_DeterministicOutput(t *testing.T) {
	tmp := t.TempDir()
	exe := buildBinary(t, "wrap-jsonapi")
	args := []string{
		"--input-dir", "testdata/input",
		"--output-dir", tmp,
		"--type", "shop",
		"--id-field", "npcId",
		"--filename-prefix", "shop",
	}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("first run: %v\n%s", err, out)
	}
	first, err := os.ReadFile(filepath.Join(tmp, "shop-1001.json"))
	if err != nil {
		t.Fatalf("read first: %v", err)
	}
	want, _ := os.ReadFile("testdata/expected/shop-1001.json")
	if !bytes.Equal(first, want) {
		t.Fatalf("output mismatch:\n--- got ---\n%s\n--- want ---\n%s", first, want)
	}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("second run: %v\n%s", err, out)
	}
	second, _ := os.ReadFile(filepath.Join(tmp, "shop-1001.json"))
	if !bytes.Equal(first, second) {
		t.Fatalf("non-deterministic: rerun differs")
	}
}

func buildBinary(t *testing.T, name string) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), name)
	out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return exe
}
