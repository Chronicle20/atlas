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

// TestWrap_LargeNumericID verifies that a 7-digit numeric id is NOT converted to
// scientific notation (e.g. "1.002002e+06"). Regression for the json.Unmarshal
// float64 coercion bug.
func TestWrap_LargeNumericID(t *testing.T) {
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
		t.Fatalf("run: %v\n%s", err, out)
	}
	got, err := os.ReadFile(filepath.Join(tmp, "shop-1002002.json"))
	if err != nil {
		t.Fatalf("expected shop-1002002.json to exist (would be shop-1.002002e+06.json with bug): %v", err)
	}
	want, _ := os.ReadFile("testdata/expected/shop-1002002.json")
	if !bytes.Equal(got, want) {
		t.Fatalf("output mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
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
