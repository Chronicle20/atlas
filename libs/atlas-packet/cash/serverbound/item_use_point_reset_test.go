package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUsePointResetRoundTrip(t *testing.T) {
	for _, utf := range []bool{true, false} {
		name := "trailingUpdateTime"
		if utf {
			name = "updateTimeFirst"
		}
		t.Run(name, func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 83, 1)
			input := ItemUsePointReset{to: 2048, from: 64, updateTime: 12345, updateTimeFirst: utf}
			output := ItemUsePointReset{updateTimeFirst: utf}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.To() != input.To() {
				t.Errorf("To: got %d want %d", output.To(), input.To())
			}
			if output.From() != input.From() {
				t.Errorf("From: got %d want %d", output.From(), input.From())
			}
			if !utf && output.UpdateTime() != input.UpdateTime() {
				t.Errorf("UpdateTime: got %d want %d", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
