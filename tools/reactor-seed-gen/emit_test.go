package main

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestEmit_GoldenBytes(t *testing.T) {
	doc := scriptDoc{
		ReactorId:   "2119000",
		Description: "Tombstone in Forest of Dead Trees I MSEA reference: http:// If the chest is destroyed before Riche, killing him should yield no exp",
		HitRules: []ruleDoc{
			{
				Id: "weaken_area_boss",
				Conditions: []condDoc{
					{Type: "reactor_state", Operator: "=", Value: "0"},
				},
				Operations: []opDoc{
					{
						Type: "weaken_area_boss",
						Params: map[string]string{
							"monsterId": "6090000",
							"message":   "As the tombstone lit up and vanished, Lich lost all his magic abilities.",
						},
					},
				},
			},
		},
		ActRules: nil,
	}

	got, err := marshalEnvelope(doc)
	if err != nil {
		t.Fatalf("marshalEnvelope: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "golden", "reactor-2119000.json"))
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("marshalEnvelope output mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestEmit_BareDropOmitsParams(t *testing.T) {
	doc := scriptDoc{
		ReactorId:   "1002008",
		Description: "Henesys box - drops items",
		ActRules: []ruleDoc{
			{
				Id: "drop_items",
				Operations: []opDoc{
					{Type: "drop_items", Params: nil},
				},
			},
		},
	}

	got, err := marshalEnvelope(doc)
	if err != nil {
		t.Fatalf("marshalEnvelope: %v", err)
	}

	s := string(got)
	if !bytes.Contains(got, []byte(`"type": "drop_items"`)) {
		t.Fatalf("expected drop_items type in output, got:\n%s", s)
	}
	if bytes.Contains(got, []byte(`"params"`)) {
		t.Fatalf("expected no params key for nil Params, got:\n%s", s)
	}
}

func TestEmit_EmptyRulesAreArraysNotNull(t *testing.T) {
	doc := scriptDoc{
		ReactorId:   "9018000",
		Description: "some description",
		HitRules:    nil,
		ActRules:    nil,
	}

	got, err := marshalEnvelope(doc)
	if err != nil {
		t.Fatalf("marshalEnvelope: %v", err)
	}

	if !bytes.Contains(got, []byte(`"actRules": []`)) {
		t.Fatalf("expected actRules to be [], got:\n%s", got)
	}
	if !bytes.Contains(got, []byte(`"hitRules": []`)) {
		t.Fatalf("expected hitRules to be [], got:\n%s", got)
	}
	if bytes.Contains(got, []byte(`null`)) {
		t.Fatalf("expected no null in output, got:\n%s", got)
	}
}

func TestEmit_ConditionOmitsEmptyStep(t *testing.T) {
	doc := scriptDoc{
		ReactorId:   "2119000",
		Description: "some description",
		HitRules: []ruleDoc{
			{
				Id: "r1",
				Conditions: []condDoc{
					{Type: "reactor_state", Operator: "=", Value: "0"},
				},
				Operations: []opDoc{{Type: "noop", Params: nil}},
			},
		},
		ActRules: []ruleDoc{
			{
				Id: "r2",
				Conditions: []condDoc{
					{Type: "pq_custom_data", Operator: "=", Value: "5", Step: "stage"},
				},
				Operations: []opDoc{{Type: "noop", Params: nil}},
			},
		},
	}

	got, err := marshalEnvelope(doc)
	if err != nil {
		t.Fatalf("marshalEnvelope: %v", err)
	}

	s := string(got)
	if n := bytes.Count(got, []byte(`"step"`)); n != 1 {
		t.Fatalf("expected exactly one \"step\" key (from the pq_custom_data condition only), got %d:\n%s", n, s)
	}
	if !bytes.Contains(got, []byte(`"step": "stage"`)) {
		t.Fatalf("pq_custom_data condition should contain step \"stage\":\n%s", s)
	}
}

func TestEmit_TrailingNewline(t *testing.T) {
	doc := scriptDoc{
		ReactorId:   "2119000",
		Description: "some description",
	}

	got, err := marshalEnvelope(doc)
	if err != nil {
		t.Fatalf("marshalEnvelope: %v", err)
	}

	if len(got) == 0 || got[len(got)-1] != '\n' {
		t.Fatalf("expected output to end with a single newline, got: %q", got)
	}
	if bytes.HasSuffix(got, []byte("\n\n")) {
		t.Fatalf("expected exactly one trailing newline, got two or more: %q", got)
	}
	if bytes.Contains(got, []byte("\r")) {
		t.Fatalf("expected no carriage returns in output, got: %q", got)
	}
}

func TestFanOut_WritesElevenIdenticalCopies(t *testing.T) {
	root := t.TempDir()
	for _, dir := range seedDirs {
		if err := os.MkdirAll(filepath.Join(root, dir, "reactor-actions", "reactors"), 0o755); err != nil {
			t.Fatalf("setting up seed dir %s: %v", dir, err)
		}
	}

	golden := []byte("{\n  \"data\": {}\n}\n")
	if err := fanOut(root, "2119000", golden); err != nil {
		t.Fatalf("fanOut: %v", err)
	}

	wantSum := sha256.Sum256(golden)

	if len(seedDirs) != 11 {
		t.Fatalf("expected 11 seed directories, got %d", len(seedDirs))
	}

	for _, dir := range seedDirs {
		path := filepath.Join(root, dir, "reactor-actions", "reactors", "reactor-2119000.json")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected file at %s: %v", path, err)
		}
		gotSum := sha256.Sum256(b)
		if gotSum != wantSum {
			t.Fatalf("file at %s has different content than golden", path)
		}
	}
}
