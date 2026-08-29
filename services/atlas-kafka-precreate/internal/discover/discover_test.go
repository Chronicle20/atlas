package discover

import (
	"reflect"
	"testing"
)

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
