package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=messenger/serverbound/MessengerOperationChat version=gms_v83 ida=0x8511fc
// packet-audit:verify packet=messenger/serverbound/MessengerOperationChat version=gms_v87 ida=0x8b978f
// packet-audit:verify packet=messenger/serverbound/MessengerOperationChat version=gms_v95 ida=0x7f6140
// packet-audit:verify packet=messenger/serverbound/MessengerOperationChat version=jms_v185 ida=0x8e4f92
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
