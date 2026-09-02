package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseConditionTypes(t *testing.T) {
	src := `package saga

const (
	JobCondition   = "jobId"
	MesoCondition  = "meso"
	LevelCondition = "level"
)

const (
	Equals = "="
	In     = "in"
)
`

	got, err := parseStringConsts(src, 0)
	if err != nil {
		t.Fatalf("parseStringConsts(block 0) failed: %v", err)
	}
	want := []string{"jobId", "meso", "level"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseStringConsts(block 0) = %v, want %v", got, want)
	}

	got, err = parseStringConsts(src, 1)
	if err != nil {
		t.Fatalf("parseStringConsts(block 1) failed: %v", err)
	}
	want = []string{"=", "in"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseStringConsts(block 1) = %v, want %v", got, want)
	}
}

func TestParseOperationCases(t *testing.T) {
	src := `package script

func (e *OperationExecutor) ExecuteOperation(f field.Model, characterId uint32, op operation.Model) error {
	switch op.Type() {
	case "field_effect":
		return nil
	case "lock_ui":
		return nil
	case "unlock_ui":
		return nil
	default:
		return nil
	}
}
`

	got, err := parseSwitchCases(src, "ExecuteOperation")
	if err != nil {
		t.Fatalf("parseSwitchCases failed: %v", err)
	}
	want := []string{"field_effect", "lock_ui", "unlock_ui"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseSwitchCases = %v, want %v", got, want)
	}
}

func TestParseSwitchCasesMissingFunc(t *testing.T) {
	src := `package script

func (e *OperationExecutor) ExecuteOperation(f field.Model, characterId uint32, op operation.Model) error {
	switch op.Type() {
	case "field_effect":
		return nil
	case "lock_ui":
		return nil
	case "unlock_ui":
		return nil
	default:
		return nil
	}
}
`

	_, err := parseSwitchCases(src, "NoSuchFunction")
	if err == nil {
		t.Fatal("parseSwitchCases: expected error, got nil")
	}
	want := "function NoSuchFunction not found"
	if err.Error() != want {
		t.Fatalf("parseSwitchCases error = %q, want %q", err.Error(), want)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	conditionTypes := []string{"map_id", "jobId", "level"}
	operators := []string{"=", ">", "in", "==", "!="}
	operations := []string{"field_effect"}
	allOf := json.RawMessage(`[
		{
			"if": { "properties": { "type": { "const": "field_effect" } } },
			"then": { "properties": { "params": { "type": "object" } } }
		}
	]`)

	got1, err := render(conditionTypes, operators, operations, allOf)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	got2, err := render(conditionTypes, operators, operations, allOf)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if string(got1) != string(got2) {
		t.Fatalf("render is not deterministic:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", got1, got2)
	}
}

func TestRenderRequiresAllOfBlockPerOperation(t *testing.T) {
	conditionTypes := []string{"map_id"}
	operators := []string{"="}
	operations := []string{"field_effect", "play_sound"}
	allOf := json.RawMessage(`[
		{
			"if": { "properties": { "type": { "const": "field_effect" } } },
			"then": { "properties": { "params": { "type": "object" } } }
		}
	]`)

	_, err := render(conditionTypes, operators, operations, allOf)
	if err == nil {
		t.Fatal("render: expected error, got nil")
	}
	want := `operation "play_sound" has no allOf param block in the schema template`
	if err.Error() != want {
		t.Fatalf("render error = %q, want %q", err.Error(), want)
	}
}
