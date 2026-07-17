package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseTripleMegaphoneRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		lines   []string
		whisper bool
	}{
		{"lines_1", []string{"only line"}, false},
		{"lines_2", []string{"line one", "line two"}, true},
		{"lines_3", []string{"line one", "line two", "line three"}, false},
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
				input := NewItemUseTripleMegaphone(updateTimeFirst)
				input.lines = tc.lines
				input.whisper = tc.whisper
				if !updateTimeFirst {
					input.updateTime = 24680
				}
				output := NewItemUseTripleMegaphone(updateTimeFirst)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if len(output.Lines()) != len(input.Lines()) {
					t.Fatalf("lines count: got %d, want %d", len(output.Lines()), len(input.Lines()))
				}
				for i := range input.lines {
					if output.Lines()[i] != input.Lines()[i] {
						t.Errorf("line %d: got %q, want %q", i, output.Lines()[i], input.Lines()[i])
					}
				}
				if output.Whisper() != input.Whisper() {
					t.Errorf("whisper: got %v, want %v", output.Whisper(), input.Whisper())
				}
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}
