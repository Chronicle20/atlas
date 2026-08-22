package discover

import (
	"reflect"
	"testing"
)

func TestFromEnviron(t *testing.T) {
	tests := []struct {
		name        string
		environ     []string
		wantPlain   []string
		wantCompact []string
	}{
		{
			name: "prefix selection",
			environ: []string{
				"COMMAND_TOPIC_CREATE_CHARACTER=cmd-char",
				"EVENT_TOPIC_CHARACTER_STATUS=evt-char",
				"PATH=/usr/bin",
				"BOOTSTRAP_SERVERS=kafka:9092",
				"KAFKA_CONSUMER_GROUP=g",
			},
			wantPlain:   []string{"cmd-char", "evt-char"},
			wantCompact: []string{},
		},
		{
			name: "empty value skipped",
			environ: []string{
				"COMMAND_TOPIC_A=",
				"EVENT_TOPIC_B=evt-b",
			},
			wantPlain:   []string{"evt-b"},
			wantCompact: []string{},
		},
		{
			name: "duplicates collapsed",
			environ: []string{
				"COMMAND_TOPIC_A=shared",
				"COMMAND_TOPIC_B=shared",
				"EVENT_TOPIC_C=shared",
			},
			wantPlain:   []string{"shared"},
			wantCompact: []string{},
		},
		{
			name: "config-status vars are compacted",
			environ: []string{
				"EVENT_TOPIC_CONFIGURATION_TENANT_STATUS=cfg-tenant",
				"EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS=cfg-service",
				"EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS=cfg-env",
				"EVENT_TOPIC_OTHER=evt-other",
			},
			wantPlain:   []string{"evt-other"},
			wantCompact: []string{"cfg-env", "cfg-service", "cfg-tenant"},
		},
		{
			name: "compaction wins on collision",
			environ: []string{
				"EVENT_TOPIC_CONFIGURATION_TENANT_STATUS=both",
				"COMMAND_TOPIC_X=both",
			},
			wantPlain:   []string{},
			wantCompact: []string{"both"},
		},
		{
			name: "underscore vs hyphen ordering is byte order",
			environ: []string{
				"COMMAND_TOPIC_A=topic_b",
				"COMMAND_TOPIC_B=topic-a",
				"COMMAND_TOPIC_C=topicZ",
			},
			wantPlain:   []string{"topic-a", "topicZ", "topic_b"},
			wantCompact: []string{},
		},
		{
			name:        "no matching vars",
			environ:     []string{"PATH=/usr/bin"},
			wantPlain:   []string{},
			wantCompact: []string{},
		},
		{
			name:        "value containing =",
			environ:     []string{"COMMAND_TOPIC_A=a=b"},
			wantPlain:   []string{"a=b"},
			wantCompact: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromEnviron(tt.environ)
			if !reflect.DeepEqual(got.Plain, tt.wantPlain) {
				t.Errorf("Plain = %q, want %q", got.Plain, tt.wantPlain)
			}
			if !reflect.DeepEqual(got.Compact, tt.wantCompact) {
				t.Errorf("Compact = %q, want %q", got.Compact, tt.wantCompact)
			}
		})
	}
}

func TestGroups(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "unset",
			raw:  "",
			want: []string{},
		},
		{
			name: "whitespace only",
			raw:  "\n  \n",
			want: []string{"  "},
		},
		{
			name: "single group",
			raw:  "Account Service [pr-123]",
			want: []string{"Account Service [pr-123]"},
		},
		{
			name: "multi-line",
			raw:  "World Service [pr-123]\nChannel Service - 3f8c [pr-123]",
			want: []string{"World Service [pr-123]", "Channel Service - 3f8c [pr-123]"},
		},
		{
			name: "blank lines dropped",
			raw:  "a\n\n\nb\n",
			want: []string{"a", "b"},
		},
		{
			name: "CRLF trimmed",
			raw:  "a\r\nb\r\n",
			want: []string{"a", "b"},
		},
		{
			name: "spaces and brackets round-trip",
			raw:  "Channel Service - 7c2f8b1e-0d4a-4a1b-9f3e-2c1d5e6f7a8b [pr-1450]",
			want: []string{"Channel Service - 7c2f8b1e-0d4a-4a1b-9f3e-2c1d5e6f7a8b [pr-1450]"},
		},
		{
			name: "leading dash",
			raw:  "-weird group",
			want: []string{"-weird group"},
		},
		{
			name: "order preserved",
			raw:  "z\na",
			want: []string{"z", "a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Groups(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Groups(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestStateIsSeedable(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{"", true},
		{"Empty", true},
		{"Dead", true},
		{"Stable", false},
		{"PreparingRebalance", false},
		{"CompletingRebalance", false},
		{"AssigningPartitions", false},
		{"SomeFutureKafkaState", false},
		{"empty", false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			if got := StateIsSeedable(tt.state); got != tt.want {
				t.Errorf("StateIsSeedable(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}
