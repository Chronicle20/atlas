package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLint_MapActionExitCode(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{name: "good tree exits zero", dir: "testdata/good", wantErr: false},
		{name: "unreplicated exits non-zero", dir: "testdata/bad/map-action-unreplicated", wantErr: true},
		{name: "unguarded spawn exits non-zero", dir: "testdata/bad/map-action-unguarded-spawn", wantErr: true},
		{name: "schema invalid exits non-zero", dir: "testdata/bad/map-action-schema-invalid", wantErr: true},
		// gms/85_1 is a genuine version root (valid CATALOG_REVISION, name
		// matches <major>_<minor>) that carries no map-actions/ directory at
		// all, while gms/83_1 and gms/84_1 both do. discoverRoots must not
		// silently miss it: this must be reported as a replication
		// violation, not escape the check.
		{name: "missing root exits non-zero", dir: "testdata/bad/map-action-missing-root", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exe := buildLint(t)
			cmd := exec.Command(exe, tt.dir)
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if tt.wantErr && err == nil {
				t.Fatalf("expected non-zero exit")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected exit 0, got %v", err)
			}
		})
	}
}

func TestCheckMapActions(t *testing.T) {
	tests := []struct {
		name      string
		root      string
		docs      []mapActionDoc
		useSchema bool
		match     func(t *testing.T, errs []string)
	}{
		{
			// testdata/good/gms/{83_1,84_1} both exist on disk; simulate the
			// document being present only in 83_1 by supplying just that one doc.
			name: "unreplicated missing from root",
			root: "testdata/good",
			docs: []mapActionDoc{
				{path: "testdata/good/gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":1}`)},
			},
			match: func(t *testing.T, errs []string) {
				want := `map-actions/onUserEnter/map-t.json: present in gms/83_1, missing from gms/84_1`
				if !containsSubstring(errs, want) {
					t.Fatalf("expected missing-from message, got %v", errs)
				}
			},
		},
		{
			name: "unreplicated differs",
			root: "testdata/good",
			docs: []mapActionDoc{
				{path: "testdata/good/gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":1}`)},
				{path: "testdata/good/gms/84_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "84_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":2}`)},
			},
			match: func(t *testing.T, errs []string) {
				want := `map-actions/onUserEnter/map-t.json: differs between gms/83_1 and gms/84_1`
				if !containsSubstring(errs, want) {
					t.Fatalf("expected differs message, got %v", errs)
				}
			},
		},
		{
			// testdata/bad/map-action-missing-root/gms/85_1 has a valid
			// CATALOG_REVISION and no map-actions/ directory whatsoever. It
			// must still be counted as a version root by discoverRoots, so
			// the doc that exists in gms/83_1 and gms/84_1 is reported
			// missing from it.
			name: "missing whole root",
			root: "testdata/bad/map-action-missing-root",
			docs: []mapActionDoc{
				{path: "testdata/bad/map-action-missing-root/gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":1}`)},
				{path: "testdata/bad/map-action-missing-root/gms/84_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "84_1", hook: "onUserEnter", id: "t", raw: []byte(`{"a":1}`)},
			},
			match: func(t *testing.T, errs []string) {
				want := `map-actions/onUserEnter/map-t.json: present in gms/83_1, missing from gms/85_1`
				if !containsSubstring(errs, want) {
					t.Fatalf("expected missing-from-whole-root message, got %v", errs)
				}
			},
		},
		{
			name: "unguarded spawn",
			root: "",
			docs: []mapActionDoc{
				{
					path: "gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t",
					raw: []byte(`{
						"data": {
							"attributes": {
								"rules": [
									{"id": "r1", "operations": [{"type": "spawn_monster", "params": {"monsterId": "1"}}]}
								]
							}
						}
					}`),
				},
			},
			match: func(t *testing.T, errs []string) {
				want := `gms/83_1/map-actions/onUserEnter/map-t.json: rule "r1" operation 1: spawn_monster requires "spawnIfAbsent": "true"`
				if !containsSubstring(errs, want) {
					t.Fatalf("expected unguarded-spawn message %q, got %v", want, errs)
				}
			},
		},
		{
			name:      "schema invalid",
			root:      "",
			useSchema: true,
			docs: []mapActionDoc{
				{
					path: "gms/83_1/map-actions/onUserEnter/map-t.json", region: "gms", version: "83_1", hook: "onUserEnter", id: "t",
					raw: []byte(`{
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
					}`),
				},
			},
			match: func(t *testing.T, errs []string) {
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaPath := ""
			if tt.useSchema {
				var ok bool
				schemaPath, ok = mapActionSchemaPath()
				if !ok {
					t.Fatalf("map-action schema not resolvable from this git checkout; mapActionSchemaPath() ok=false")
				}
			}
			errs := checkMapActions(tt.root, tt.docs, schemaPath)
			tt.match(t, errs)
		})
	}
}

func TestDiscoverRoots(t *testing.T) {
	tests := []struct {
		name string
	}{
		// A root whose version segment does not match <major>_<minor>
		// (deploy/seed/shared/all's "all") is not a map-action version root
		// and must not be reported, even though it carries a valid
		// CATALOG_REVISION.
		{name: "excludes shared version-agnostic root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
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

// TestLint_SchemaNotFoundSkipsOnlySchemaCheck exercises the skip path
// mapActionSchemaPath()'s ok=false return drives: running from a cwd
// outside any git checkout must print the "schema not found" stderr note
// and still exit 0 on an otherwise-good tree, because
// checkMapActionSchema returns nil for an empty schemaPath while
// replication and spawn-guard checks still run.
func TestLint_SchemaNotFoundSkipsOnlySchemaCheck(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "non-git cwd skips only the schema check"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
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
