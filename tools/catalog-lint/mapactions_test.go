package main

import (
	"os"
	"os/exec"
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
		t.Skip("map-action schema not found in this checkout")
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

func containsSubstring(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}
