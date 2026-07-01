package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestPinOperationV72 pins the gms_v72 CHECK_PINCODE (op 6) clientbound wire.
// IDA-verified (GMS_v72.1_U_DEVM.exe, port 13339) — CLogin::OnCheckPinCodeResult
// @0x5b56b9 reads a single CInPacket::Decode1(mode) off the wire; the remainder
// is client-side switch logic. atlas PinOperation.Encode writes WriteByte(mode).
// Mode-only, 1 byte.
func TestPinOperationV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := PinOperation{mode: 3}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, []byte{0x03}) {
		t.Errorf("v72 PinOperation: got % x, want 03", got)
	}
}

// packet-audit:verify packet=login/clientbound/PinOperation version=gms_v83 ida=0x5fc89d
// packet-audit:verify packet=login/clientbound/PinOperation version=gms_v87 ida=0x6342b0
// packet-audit:verify packet=login/clientbound/PinOperation version=gms_v95 ida=0x5db000
// packet-audit:verify packet=login/clientbound/PinOperation version=gms_v84 ida=0x611975
// packet-audit:verify packet=login/clientbound/PinOperation version=gms_v79 ida=0x5d0921
// packet-audit:verify packet=login/clientbound/PinOperation version=gms_v72 ida=0x5b56b9
func TestPinOperationRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := PinOperation{mode: 3}
			output := PinOperation{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
