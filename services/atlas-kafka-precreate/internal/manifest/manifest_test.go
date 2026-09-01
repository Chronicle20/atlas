package manifest

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Manifest
		wantErr string
	}{
		{
			name:  "well formed",
			input: "topics:\n  - token: COMMAND_TOPIC_A\n    cleanup: delete\n",
			want: Manifest{
				Topics: []Entry{{Token: "COMMAND_TOPIC_A", Cleanup: "delete"}},
			},
		},
		{
			name:    "malformed yaml",
			input:   "topics: [",
			wantErr: "parsing topic manifest",
		},
		{
			name:    "empty document",
			input:   "",
			wantErr: "topic manifest is empty",
		},
		{
			name:    "no topics key",
			input:   "other: 1\n",
			wantErr: "topic manifest is empty",
		},
		{
			name:    "unknown cleanup value",
			input:   "topics:\n  - token: A\n    cleanup: squash\n",
			wantErr: "squash",
		},
		{
			name:    "missing cleanup",
			input:   "topics:\n  - token: A\n",
			wantErr: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse([]byte(tt.input))
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Parse() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Parse() error = %q, want containing %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		manifest    Manifest
		env         map[string]string
		wantPlain   []string
		wantCompact []string
		wantErr     string
	}{
		{
			name: "plain and compact",
			manifest: Manifest{Topics: []Entry{
				{Token: "A", Cleanup: "delete"},
				{Token: "B", Cleanup: "compact"},
			}},
			env:         map[string]string{"A": "a", "B": "b"},
			wantPlain:   []string{"a"},
			wantCompact: []string{"b"},
		},
		{
			name: "duplicates collapsed",
			manifest: Manifest{Topics: []Entry{
				{Token: "A", Cleanup: "delete"},
				{Token: "B", Cleanup: "delete"},
			}},
			env:         map[string]string{"A": "shared", "B": "shared"},
			wantPlain:   []string{"shared"},
			wantCompact: []string{},
		},
		{
			name: "compaction wins on collision",
			manifest: Manifest{Topics: []Entry{
				{Token: "A", Cleanup: "delete"},
				{Token: "B", Cleanup: "compact"},
			}},
			env:         map[string]string{"A": "both", "B": "both"},
			wantPlain:   []string{},
			wantCompact: []string{"both"},
		},
		{
			name: "sorted output",
			manifest: Manifest{Topics: []Entry{
				{Token: "A", Cleanup: "delete"},
				{Token: "B", Cleanup: "delete"},
				{Token: "C", Cleanup: "delete"},
			}},
			env:         map[string]string{"A": "topic_b", "B": "topic-a", "C": "topicZ"},
			wantPlain:   []string{"topic-a", "topicZ", "topic_b"},
			wantCompact: []string{},
		},
		{
			name: "unresolved token is fatal",
			manifest: Manifest{Topics: []Entry{
				{Token: "A", Cleanup: "delete"},
				{Token: "B", Cleanup: "delete"},
			}},
			env:     map[string]string{"A": "a"},
			wantErr: "topic manifest token [B] has no value in the environment",
		},
		{
			name: "empty value is fatal",
			manifest: Manifest{Topics: []Entry{
				{Token: "A", Cleanup: "delete"},
			}},
			env:     map[string]string{"A": ""},
			wantErr: "topic manifest token [A] has no value in the environment",
		},
		{
			name: "first unresolved by sort order",
			manifest: Manifest{Topics: []Entry{
				{Token: "Z", Cleanup: "delete"},
				{Token: "A", Cleanup: "delete"},
			}},
			env:     map[string]string{},
			wantErr: "topic manifest token [A] has no value in the environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			look := func(token string) (string, bool) {
				value, ok := tt.env[token]
				return value, ok
			}

			got, err := Resolve(tt.manifest, look)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Resolve() error = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("Resolve() error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got.Plain, tt.wantPlain) {
				t.Errorf("Plain = %q, want %q", got.Plain, tt.wantPlain)
			}
			if !reflect.DeepEqual(got.Compact, tt.wantCompact) {
				t.Errorf("Compact = %q, want %q", got.Compact, tt.wantCompact)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "topics.yaml")
		document := "topics:\n  - token: COMMAND_TOPIC_A\n    cleanup: delete\n"
		if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		want, err := Parse([]byte(document))
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}

		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Load() = %+v, want %+v", got, want)
		}
	})

	t.Run("nonexistent path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.yaml")
		_, err := Load(path)
		if err == nil {
			t.Fatalf("Load() error = nil, want error containing %q", path)
		}
		if !strings.Contains(err.Error(), path) {
			t.Fatalf("Load() error = %q, want containing %q", err.Error(), path)
		}
	})
}
