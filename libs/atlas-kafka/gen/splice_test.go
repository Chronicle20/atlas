package main

import (
	"strings"
	"testing"
)

func TestSplice(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		block      string
		want       string
		wantErrHas string
	}{
		{
			name:  "replaces marked region",
			input: "a\n# B\nold\n# E\nz\n",
			block: "new\n",
			want:  "a\n# B\nnew\n# E\nz\n",
		},
		{
			name:  "preserves CRLF outside markers",
			input: "a\r\n# B\r\nold\r\n# E\r\nz\r\n",
			block: "new\r\n",
			want:  "a\r\n# B\r\nnew\r\n# E\r\nz\r\n",
		},
		{
			name:  "empty block",
			input: "a\n# B\nold\n# E\n",
			block: "",
			want:  "a\n# B\n# E\n",
		},
		{
			name:       "missing begin marker",
			input:      "a\nold\n# E\n",
			block:      "new\n",
			wantErrHas: "# B",
		},
		{
			name:       "missing end marker",
			input:      "a\n# B\nold\n",
			block:      "new\n",
			wantErrHas: "# E",
		},
		{
			name:       "end before begin",
			input:      "# E\nx\n# B\n",
			block:      "new\n",
			wantErrHas: "out of order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Splice([]byte(tt.input), "# B", "# E", []byte(tt.block))
			if tt.wantErrHas != "" {
				if err == nil {
					t.Fatalf("Splice() error = nil, want error containing %q", tt.wantErrHas)
				}
				if !strings.Contains(err.Error(), tt.wantErrHas) {
					t.Fatalf("Splice() error = %q, want it to contain %q", err.Error(), tt.wantErrHas)
				}
				return
			}
			if err != nil {
				t.Fatalf("Splice() unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Splice() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmitEnvConfigMapBlock(t *testing.T) {
	m := Manifest{Topics: []Entry{
		{Token: "COMMAND_TOPIC_A", Cleanup: "delete"},
		{Token: "EVENT_TOPIC_B", Cleanup: "delete"},
	}}

	got := string(m.EmitEnvConfigMapBlock())
	want := "  COMMAND_TOPIC_A: \"COMMAND_TOPIC_A\"\n  EVENT_TOPIC_B: \"EVENT_TOPIC_B\"\n"
	if got != want {
		t.Fatalf("EmitEnvConfigMapBlock() = %q, want %q", got, want)
	}
}

func TestEmitTopicsConfigMap(t *testing.T) {
	m := Manifest{Topics: []Entry{
		{Token: "COMMAND_TOPIC_A", Cleanup: "delete", Packages: []string{"example.com/a"}},
		{Token: "EVENT_TOPIC_B", Cleanup: "compact", Packages: []string{"example.com/b"}},
	}}

	got := string(m.EmitTopicsConfigMap())

	if !strings.Contains(got, "name: atlas-kafka-topics") {
		t.Fatalf("EmitTopicsConfigMap() missing ConfigMap name, got:\n%s", got)
	}
	if !strings.Contains(got, `argocd.argoproj.io/sync-wave: "-1"`) {
		t.Fatalf("EmitTopicsConfigMap() missing sync-wave annotation, got:\n%s", got)
	}
	if !strings.Contains(got, "topics.yaml:") {
		t.Fatalf("EmitTopicsConfigMap() missing topics.yaml key, got:\n%s", got)
	}
	if !strings.Contains(got, "token: COMMAND_TOPIC_A") || !strings.Contains(got, "cleanup: delete") {
		t.Fatalf("EmitTopicsConfigMap() missing token/cleanup pair, got:\n%s", got)
	}
	if !strings.Contains(got, "token: EVENT_TOPIC_B") || !strings.Contains(got, "cleanup: compact") {
		t.Fatalf("EmitTopicsConfigMap() missing second token/cleanup pair, got:\n%s", got)
	}
	if strings.Contains(got, "packages:") {
		t.Fatalf("EmitTopicsConfigMap() must not carry provenance packages, got:\n%s", got)
	}
}
