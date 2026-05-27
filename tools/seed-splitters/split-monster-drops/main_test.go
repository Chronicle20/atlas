package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSplitMonsterDrops_MatchesExpected(t *testing.T) {
	tmp := t.TempDir()
	exe := filepath.Join(t.TempDir(), "split")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	args := []string{"--input", "testdata/input/monster_drops.json", "--output", tmp}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	for _, name := range []string{"monster-100.json", "monster-200.json"} {
		got, _ := os.ReadFile(filepath.Join(tmp, name))
		want, _ := os.ReadFile(filepath.Join("testdata/expected", name))
		if !bytes.Equal(got, want) {
			t.Fatalf("%s mismatch:\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
		}
	}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("rerun: %v\n%s", err, out)
	}
	got, _ := os.ReadFile(filepath.Join(tmp, "monster-100.json"))
	want, _ := os.ReadFile(filepath.Join("testdata/expected", "monster-100.json"))
	if !bytes.Equal(got, want) {
		t.Fatalf("rerun produced different output")
	}
}
