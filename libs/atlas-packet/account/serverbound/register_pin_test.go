package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61: REGISTER_PIN send built in CLogin::OnCheckPinCodeResult @0x5688ce
// (register arm @0x568b31): Encode1(pinInput flag)@0x568b59 + EncodeStr(pin)
// @0x568b9a when set. Matches atlas RegisterPin.Encode. pinInput=true,pin="1234"
// → 01 04 00 '1''2''3''4'.
//
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v61 ida=0x5688ce
func TestRegisterPinV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := RegisterPin{pinInput: true, pin: "1234"}
	want := []byte{0x01, 0x04, 0x00, '1', '2', '3', '4'} // pinInput bool + len-prefixed pin
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 RegisterPin body: got % x, want % x", got, want)
	}
}

// gms_v72: REGISTER_PIN send built in CLogin::OnCheckPinCodeResult @0x5b56b9
// (#RegisterPin arm) — Encode1(pinInput flag) + EncodeStr(pin); same shape as
// v79 (GMS_v72.1_U_DEVM.exe, port 13339). Matches atlas RegisterPin.Encode.
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v72 ida=0x5b56b9
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v83 ida=0x5fc89d
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v87 ida=0x6342b0
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v95 ida=0x5db000
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v84 ida=0x611975
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v79 ida=0x5d0921
func TestRegisterPinRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name+"/with_pin", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := RegisterPin{pinInput: true, pin: "1234"}
			output := RegisterPin{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PinInput() != input.PinInput() {
				t.Errorf("pinInput: got %v, want %v", output.PinInput(), input.PinInput())
			}
			if output.Pin() != input.Pin() {
				t.Errorf("pin: got %v, want %v", output.Pin(), input.Pin())
			}
		})
		t.Run(v.Name+"/no_pin", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := RegisterPin{pinInput: false}
			output := RegisterPin{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PinInput() != input.PinInput() {
				t.Errorf("pinInput: got %v, want %v", output.PinInput(), input.PinInput())
			}
		})
	}
}
