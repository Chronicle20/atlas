package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseAvatarMegaphoneRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		whisper bool
	}{
		{"whisper_false", false},
		{"whisper_true", true},
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
				input := NewItemUseAvatarMegaphone(updateTimeFirst)
				input.lines = [4]string{"a1", "a2", "a3", "a4"}
				input.whisper = tc.whisper
				if !updateTimeFirst {
					input.updateTime = 13579
				}
				output := NewItemUseAvatarMegaphone(updateTimeFirst)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
