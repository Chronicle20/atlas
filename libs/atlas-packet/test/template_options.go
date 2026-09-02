package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// templateDir is the seed-template directory, relative to the repo root.
var templateDir = filepath.Join("services", "atlas-configurations", "seed-data", "templates")

// TemplateWriterOptions returns the options map a seed template registers a
// writer with — the very map atlas-channel hands that writer's Encode at
// runtime.
//
// Driving an encoder test from the shipped template, rather than from a
// hand-built table, is what makes the test able to see a MISSING one. A codec
// that silently degrades when options are absent (model.ReserializeMovePath
// bails out and rebroadcasts the capture verbatim; NormalElement.Encode's
// FALL_DOWN check never fires) looks perfect in a test that always supplies the
// table by hand, and shipped broken anyway — which is exactly what happened to
// the summon/dragon move-path fix.
//
// The writer must exist in the template and must carry options: an absent
// entry fails the test rather than skipping it, so a template edit that drops
// the registration cannot quietly turn this into a no-op.
func TemplateWriterOptions(t *testing.T, templateFile string, writer string) map[string]interface{} {
	t.Helper()

	path := filepath.Join(repoRoot(t), templateDir, templateFile)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read seed template %s: %v", path, err)
	}

	var doc struct {
		Socket struct {
			Writers []struct {
				Writer  string                 `json:"writer"`
				OpCode  string                 `json:"opCode"`
				Options map[string]interface{} `json:"options"`
			} `json:"writers"`
		} `json:"socket"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse seed template %s: %v", path, err)
	}

	var found int
	var options map[string]interface{}
	for _, w := range doc.Socket.Writers {
		if w.Writer == writer {
			found++
			options = w.Options
		}
	}
	if found != 1 {
		t.Fatalf("%s registers writer %q %d times, want exactly 1", templateFile, writer, found)
	}
	return options
}

// repoRoot walks up from the test's working directory (go test runs a test in
// its own package directory) to the directory that holds the seed templates.
func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, templateDir)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no directory containing %s above the working directory", templateDir)
		}
		dir = parent
	}
}
