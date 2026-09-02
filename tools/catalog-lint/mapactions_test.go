package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLint_MapActionGoodTreeExitsZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/good")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("expected exit 0, got %v", err)
	}
}

func TestLint_MapActionUnreplicatedExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/map-action-unreplicated")
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

func TestLint_MapActionUnguardedSpawnExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/map-action-unguarded-spawn")
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

func TestLint_MapActionSchemaInvalidExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/map-action-schema-invalid")
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

// TestLint_MapActionMissingRootExitsNonZero covers the case discoverRoots
// must not silently miss: gms/85_1 is a genuine version root (valid
// CATALOG_REVISION, name matches <major>_<minor>) that carries no
// map-actions/ directory at all, while gms/83_1 and gms/84_1 both do. This
// must be reported as a replication violation, not escape the check.
func TestLint_MapActionMissingRootExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/map-action-missing-root")
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

func TestCheckMapActions_UnreplicatedMissingFromRoot(t *testing.T) {
	// testdata/good/gms/{83_1,84_1} both exist on disk; simulate the
	// document being present only in 83_1 by supplying just that one doc.
	root := "testdata/good"
	docs := []mapActionDoc{
		{path: root + "/gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":1}`)},
	}
	errs := checkMapActions(root, docs, "")
	if !containsSubstring(errs, `map-actions/onUserEnter/map-t.json: present in gms/83_1, missing from gms/84_1`) {
		t.Fatalf("expected missing-from message, got %v", errs)
	}
}

func TestCheckMapActions_UnreplicatedDiffers(t *testing.T) {
	root := "testdata/good"
	docs := []mapActionDoc{
		{path: root + "/gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":1}`)},
		{path: root + "/gms/84_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "84_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":2}`)},
	}
	errs := checkMapActions(root, docs, "")
	if !containsSubstring(errs, `map-actions/onUserEnter/map-t.json: differs between gms/83_1 and gms/84_1`) {
		t.Fatalf("expected differs message, got %v", errs)
	}
}

// TestCheckMapActions_MissingWholeRoot exercises discoverRoots directly:
// testdata/bad/map-action-missing-root/gms/85_1 has a valid CATALOG_REVISION
// and no map-actions/ directory whatsoever. It must still be counted as a
// version root by discoverRoots, so the doc that exists in gms/83_1 and
// gms/84_1 is reported missing from it.
func TestCheckMapActions_MissingWholeRoot(t *testing.T) {
	root := "testdata/bad/map-action-missing-root"
	docs := []mapActionDoc{
		{path: root + "/gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":1}`)},
		{path: root + "/gms/84_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "84_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":1}`)},
	}
	errs := checkMapActions(root, docs, "")
	if !containsSubstring(errs, `map-actions/onUserEnter/map-t.json: present in gms/83_1, missing from gms/85_1`) {
		t.Fatalf("expected missing-from-whole-root message, got %v", errs)
	}
}

// TestDiscoverRoots_ExcludesSharedVersionAgnosticRoot documents the other
// direction: a root whose version segment does not match <major>_<minor>
// (deploy/seed/shared/all's "all") is not a map-action version root and
// must not be reported, even though it carries a valid CATALOG_REVISION.
func TestDiscoverRoots_ExcludesSharedVersionAgnosticRoot(t *testing.T) {
	tmp := t.TempDir()
	mustWriteFile(t, filepath.Join(tmp, "shared", "all", "CATALOG_REVISION"), "rev")
	mustWriteFile(t, filepath.Join(tmp, "gms", "83_1", "CATALOG_REVISION"), "rev")

	roots := discoverRoots(tmp)
	for _, r := range roots {
		if r == "shared/all" {
			t.Fatalf("expected shared/all to be excluded, got roots %v", roots)
		}
	}
	if !containsStringSlice(roots, "gms/83_1") {
		t.Fatalf("expected gms/83_1 to be discovered, got roots %v", roots)
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func containsStringSlice(s []string, v string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

func TestCheckMapActions_UnguardedSpawn(t *testing.T) {
	raw := []byte(`{
		"data": {
			"attributes": {
				"rules": [
					{"id": "r1", "operations": [{"type": "spawn_monster", "params": {"monsterId": "1"}}]}
				]
			}
		}
	}`)
	docs := []mapActionDoc{
		{path: "gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t", raw: raw},
	}
	errs := checkMapActions("", docs, "")
	want := `gms/83_1/map-actions/onUserEnter/map-t.json: rule "r1" operation 1: spawn_monster requires "spawnIfAbsent": "true"`
	if !containsSubstring(errs, want) {
		t.Fatalf("expected unguarded-spawn message %q, got %v", want, errs)
	}
}

func TestCheckMapActions_SchemaInvalid(t *testing.T) {
	schemaPath, ok := mapActionSchemaPath()
	if !ok {
		t.Fatalf("map-action schema not resolvable from this git checkout; mapActionSchemaPath() ok=false")
	}
	raw := []byte(`{
		"data": {
			"attributes": {
				"scriptName": "t",
				"rules": [
					{
						"id": "r1",
						"conditions": [{"type": "quest_status", "operator": "=", "value": "1"}],
						"operations": [{"type": "spawn_monster", "params": {"monsterId": "1", "spawnIfAbsent": "true", "x": "0", "y": "0"}}]
					}
				]
			}
		}
	}`)
	docs := []mapActionDoc{
		{path: "gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t", raw: raw},
	}
	errs := checkMapActions("", docs, schemaPath)

	var found string
	for _, e := range errs {
		if strings.Contains(e, "schema:") && strings.Contains(e, "/rules/0/conditions/0/type") {
			found = e
			break
		}
	}
	if found == "" {
		t.Fatalf("expected a schema error mentioning schema: and the JSON pointer, got %v", errs)
	}
}

// TestLint_SchemaNotFoundSkipsOnlySchemaCheck exercises the skip path
// mapActionSchemaPath()'s ok=false return drives: running from a cwd
// outside any git checkout must print the "schema not found" stderr note
// and still exit 0 on an otherwise-good tree, because
// checkMapActionSchema returns nil for an empty schemaPath while
// replication and spawn-guard checks still run.
func TestLint_SchemaNotFoundSkipsOnlySchemaCheck(t *testing.T) {
	exe := buildLint(t)
	goodTreeAbs, err := filepath.Abs("testdata/good")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}

	nonGitDir := t.TempDir()
	if _, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		// Sanity-check the harness's own assumption: t.TempDir() must not
		// itself resolve to a git checkout, or this test would pass for
		// the wrong reason.
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = nonGitDir
		if out, err := cmd.Output(); err == nil {
			t.Fatalf("t.TempDir() %s unexpectedly resolves to a git checkout %q; cannot exercise the non-git skip path", nonGitDir, strings.TrimSpace(string(out)))
		}
	}

	cmd := exec.Command(exe, goodTreeAbs)
	cmd.Dir = nonGitDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("expected exit 0 with schema validation skipped, got %v (stderr: %s)", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "map-action schema not found") {
		t.Fatalf("expected stderr to note the skipped schema check, got %q", stderr.String())
	}
}

func containsSubstring(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}
