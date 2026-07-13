package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestRegisterPinV48Body pins the gms_v48 REGISTER_PIN (COutPacket op 9)
// serverbound wire.
//
// IDA-verified (GMS_v48_1_DEVM.exe, port 13337) — CLogin::OnCheckPinCodeResult
// = sub_503956 @0x503956. The Decode1(dialogResult)==1 arm @0x503bb8 builds
// COutPacket(9): on success (sub_519081 >= 0) Encode1(1)@0x503be0 +
// EncodeStr(pin)@0x503c21; on failure Encode1(0)@0x503bc9. Matches atlas
// RegisterPin.Encode. pinInput=true, pin="1234" → 01 04 00 '1''2''3''4'.
//
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v48 ida=0x503956
func TestRegisterPinV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := RegisterPin{pinInput: true, pin: "1234"}
	want := []byte{0x01, 0x04, 0x00, '1', '2', '3', '4'} // pinInput bool + len-prefixed pin
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 RegisterPin body: got % x, want % x", got, want)
	}
}
