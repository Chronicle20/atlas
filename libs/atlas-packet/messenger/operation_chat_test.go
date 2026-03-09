package messenger

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationChatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationChat{msg: "Hello messenger!"}
			output := OperationChat{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Msg() != input.Msg() {
				t.Errorf("msg: got %v, want %v", output.Msg(), input.Msg())
			}
		})
	}
}
