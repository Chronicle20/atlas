package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseStringConsts(t *testing.T) {
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

	tests := []struct {
		name  string
		block int
		want  []string
	}{
		{name: "block 0", block: 0, want: []string{"jobId", "meso", "level"}},
		{name: "block 1", block: 1, want: []string{"=", "in"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStringConsts(src, tt.block)
			if err != nil {
				t.Fatalf("parseStringConsts(block %d) failed: %v", tt.block, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseStringConsts(block %d) = %v, want %v", tt.block, got, tt.want)
			}
		})
	}
}

func TestParseSwitchCases(t *testing.T) {
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

	tests := []struct {
		name     string
		funcName string
		want     []string
		wantErr  string
	}{
		{
			name:     "existing function",
			funcName: "ExecuteOperation",
			want:     []string{"field_effect", "lock_ui", "unlock_ui"},
		},
		{
			name:     "missing function",
			funcName: "NoSuchFunction",
			wantErr:  "function NoSuchFunction not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSwitchCases(src, tt.funcName)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("parseSwitchCases: expected error, got nil")
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("parseSwitchCases error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSwitchCases failed: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseSwitchCases = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name           string
		conditionTypes []string
		operators      []string
		operations     []string
		allOf          json.RawMessage
		check          func(t *testing.T, conditionTypes, operators, operations []string, allOf json.RawMessage)
	}{
		{
			name:           "deterministic across repeated renders",
			conditionTypes: []string{"map_id", "jobId", "level"},
			operators:      []string{"=", ">", "in", "==", "!="},
			operations:     []string{"field_effect"},
			allOf: json.RawMessage(`[
				{
					"if": { "properties": { "type": { "const": "field_effect" } } },
					"then": { "properties": { "params": { "type": "object" } } }
				}
			]`),
			check: func(t *testing.T, conditionTypes, operators, operations []string, allOf json.RawMessage) {
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
			},
		},
		{
			name:           "requires an allOf block per operation",
			conditionTypes: []string{"map_id"},
			operators:      []string{"="},
			operations:     []string{"field_effect", "play_sound"},
			allOf: json.RawMessage(`[
				{
					"if": { "properties": { "type": { "const": "field_effect" } } },
					"then": { "properties": { "params": { "type": "object" } } }
				}
			]`),
			check: func(t *testing.T, conditionTypes, operators, operations []string, allOf json.RawMessage) {
				_, err := render(conditionTypes, operators, operations, allOf)
				if err == nil {
					t.Fatal("render: expected error, got nil")
				}
				want := `operation "play_sound" has no allOf param block in the schema template`
				if err.Error() != want {
					t.Fatalf("render error = %q, want %q", err.Error(), want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.conditionTypes, tt.operators, tt.operations, tt.allOf)
		})
	}
}
