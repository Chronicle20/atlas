package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
			input := NewItemUseMegaphone(updateTimeFirst)
			input.message = "Hello world!"
			if !updateTimeFirst {
				input.updateTime = 12345
			}
			output := NewItemUseMegaphone(updateTimeFirst)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() {
				t.Errorf("message: got %q, want %q", output.Message(), input.Message())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
