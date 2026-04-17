package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestOperationSendRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationSend{toName: "Recipient", message: "Hello there!"}
			output := OperationSend{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ToName() != input.ToName() {
				t.Errorf("toName: got %v, want %v", output.ToName(), input.ToName())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}
