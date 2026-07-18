package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v79 ida=0x522c87
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v95 ida=0x539710
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v87 ida=0x55f2a3
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v83 ida=0x53792e
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=jms_v185 ida=0x574b67
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v84 ida=0x543c2c
func TestOperationFieldAddToBlackListRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationFieldAddToBlackList{name: "BlockedUser"}
			output := OperationFieldAddToBlackList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}

// TestOperationFieldAddToBlackListV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CField::AddBlackList (sub_51BBAD): Encode1(0x1D)=mode @0x51bbd1 then EncodeStr(name) @0x51bbeb. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v72 ida=0x51bbad
func TestOperationFieldAddToBlackListV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationFieldAddToBlackList{name: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "02006869" {
		t.Errorf("v72 bytes: got %s, want 02006869", got)
	}
}
