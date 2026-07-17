package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseSuperMegaphoneRoundTrip(t *testing.T) {
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
				input := NewItemUseSuperMegaphone(updateTimeFirst)
				input.message = "Super hello!"
				input.whisper = tc.whisper
				if !updateTimeFirst {
					input.updateTime = 54321
				}
				output := NewItemUseSuperMegaphone(updateTimeFirst)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Message() != input.Message() {
					t.Errorf("message: got %q, want %q", output.Message(), input.Message())
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
