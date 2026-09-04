package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const schemaPath = "../../services/atlas-reactor-actions/docs/reactor_script_schema.json"

func buildLint(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "reactor-seed-lint")
	out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return exe
}

func TestLint(t *testing.T) {
	exe := buildLint(t)

	tests := []struct {
		name          string
		root          func(t *testing.T) string
		wantExitOK    bool
		wantSubstr    []string
		wantAnySubstr []string
		wantNoSubstr  []string
	}{
		{
			name:       "GoodCorpusExitsZero",
			root:       func(t *testing.T) string { return "testdata/good" },
			wantExitOK: true,
		},
		{
			name:          "LegacyKeysExitNonZero",
			root:          func(t *testing.T) string { return "testdata/bad/legacy-keys" },
			wantExitOK:    false,
			wantAnySubstr: []string{"mesoRange", "additionalProperties"},
		},
		{
			name:       "TypeMismatchExitsNonZero",
			root:       func(t *testing.T) string { return "testdata/bad/type-mismatch" },
			wantExitOK: false,
			wantSubstr: []string{"type"},
		},
		{
			name:       "IDMismatchExitsNonZero",
			root:       func(t *testing.T) string { return "testdata/bad/id-mismatch" },
			wantExitOK: false,
		},
		{
			name:       "MissingRequiredExitsNonZero",
			root:       func(t *testing.T) string { return "testdata/bad/missing-required" },
			wantExitOK: false,
		},
		{
			name:       "MissingDescriptionExitsNonZero",
			root:       func(t *testing.T) string { return "testdata/bad/no-description" },
			wantExitOK: false,
			wantSubstr: []string{"description"},
		},
		{
			name:       "DivergentCopiesExitNonZero",
			root:       func(t *testing.T) string { return "testdata/bad/divergent-copies" },
			wantExitOK: false,
			wantSubstr: []string{"2001"},
		},
		{
			name:       "MissingCopyExitsNonZero",
			root:       func(t *testing.T) string { return "testdata/bad/missing-copy" },
			wantExitOK: false,
		},
		{
			name:       "EmptyRootExitsNonZero",
			root:       func(t *testing.T) string { return t.TempDir() },
			wantExitOK: false,
			wantSubstr: []string{"no reactor-*.json files found"},
		},
		{
			name:         "NonexistentRootExitsNonZero",
			root:         func(t *testing.T) string { return filepath.Join(t.TempDir(), "does-not-exist") },
			wantExitOK:   false,
			wantNoSubstr: []string{"nil pointer", "panic"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(exe, tt.root(t), schemaPath)
			out, err := cmd.CombinedOutput()

			if tt.wantExitOK {
				if err != nil {
					t.Fatalf("expected exit 0, got %v\n%s", err, out)
				}
			} else {
				if err == nil {
					t.Fatalf("expected non-zero exit, got exit 0\n%s", out)
				}
			}

			for _, substr := range tt.wantSubstr {
				if !strings.Contains(string(out), substr) {
					t.Fatalf("expected output to contain %q, got:\n%s", substr, out)
				}
			}

			if len(tt.wantAnySubstr) > 0 {
				found := false
				for _, substr := range tt.wantAnySubstr {
					if strings.Contains(string(out), substr) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected output to contain one of %v, got:\n%s", tt.wantAnySubstr, out)
				}
			}

			for _, substr := range tt.wantNoSubstr {
				if strings.Contains(string(out), substr) {
					t.Fatalf("expected output not to contain %q, got:\n%s", substr, out)
				}
			}
		})
	}
}
