package session

import "testing"

func TestParsePacketWriteLog(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantAll bool
		want    []string
		absent  []string
	}{
		{
			name:   "unset disables logging",
			raw:    "",
			absent: []string{"StatChanged"},
		},
		{
			name:   "whitespace only disables logging",
			raw:    "   ",
			absent: []string{"StatChanged"},
		},
		{
			name:    "star enables every writer",
			raw:     "*",
			wantAll: true,
		},
		{
			name:   "single writer",
			raw:    "StatChanged",
			want:   []string{"StatChanged"},
			absent: []string{"CharacterBuffCancel"},
		},
		{
			name:   "comma separated list with padding",
			raw:    " StatChanged , CharacterBuffCancel ",
			want:   []string{"StatChanged", "CharacterBuffCancel"},
			absent: []string{"CharacterBuffGive"},
		},
		{
			name:   "empty entries are dropped",
			raw:    "StatChanged,,",
			want:   []string{"StatChanged"},
			absent: []string{""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			all, set := parsePacketWriteLog(tc.raw)
			if all != tc.wantAll {
				t.Fatalf("all = %v, want %v", all, tc.wantAll)
			}
			for _, name := range tc.want {
				if _, ok := set[name]; !ok {
					t.Errorf("writer %q missing from set", name)
				}
			}
			for _, name := range tc.absent {
				if _, ok := set[name]; ok {
					t.Errorf("writer %q unexpectedly present in set", name)
				}
			}
			if !tc.wantAll && len(tc.want) != len(set) {
				t.Errorf("set size = %d, want %d", len(set), len(tc.want))
			}
		})
	}
}
