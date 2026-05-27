package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSplitGachapons_MatchesExpected(t *testing.T) {
	tmp := t.TempDir()
	exe := filepath.Join(t.TempDir(), "split")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	args := []string{
		"--gachapons", "testdata/input/gachapons.json",
		"--items", "testdata/input/gachapon_items.json",
		"--global", "testdata/input/global_gachapon_items.json",
		"--output", tmp,
	}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	for _, rel := range []string{"gachapon-henesys.json", "gachapon-ellinia.json", "_global/items.json"} {
		got, _ := os.ReadFile(filepath.Join(tmp, rel))
		want, _ := os.ReadFile(filepath.Join("testdata/expected", rel))
		if !bytes.Equal(got, want) {
			t.Fatalf("%s mismatch:\n--- got ---\n%s\n--- want ---\n%s", rel, got, want)
		}
	}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("rerun: %v\n%s", err, out)
	}
	got, _ := os.ReadFile(filepath.Join(tmp, "gachapon-henesys.json"))
	want, _ := os.ReadFile(filepath.Join("testdata/expected", "gachapon-henesys.json"))
	if !bytes.Equal(got, want) {
		t.Fatalf("rerun produced different output")
	}
}
