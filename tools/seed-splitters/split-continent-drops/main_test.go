package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSplitContinentDrops_MatchesExpected(t *testing.T) {
	tmp := t.TempDir()
	exe := filepath.Join(t.TempDir(), "split")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	args := []string{"--input", "testdata/input/continent_drops.json", "--output", tmp}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	for _, name := range []string{"continent--1.json", "continent-100.json"} {
		got, _ := os.ReadFile(filepath.Join(tmp, name))
		want, _ := os.ReadFile(filepath.Join("testdata/expected", name))
		if !bytes.Equal(got, want) {
			t.Fatalf("%s mismatch:\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
		}
	}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("rerun: %v\n%s", err, out)
	}
	got, _ := os.ReadFile(filepath.Join(tmp, "continent--1.json"))
	want, _ := os.ReadFile(filepath.Join("testdata/expected", "continent--1.json"))
	if !bytes.Equal(got, want) {
		t.Fatalf("rerun produced different output")
	}
}
