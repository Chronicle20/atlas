package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v95 ida=0x496520
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v87 ida=0x484cc5
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v83 ida=0x47959e
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=jms_v185 ida=0x48bdc8
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v84 ida=0x47c73c
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
