package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61: GENDER_DONE — CLogin::OnSetAccountResult @0x56874d (GMS_v61.1_U_DEVM.exe,
// port 13338): Decode1(gender)@0x568766, Decode1(success)@0x568768; matches atlas
// SetAccountResult (gender byte + success bool). gender=1,success=true → 01 01.
//
// packet-audit:verify packet=login/clientbound/SetAccountResult version=gms_v61 ida=0x56874d
func TestSetAccountResultV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := SetAccountResult{gender: 1, success: true}
	want := []byte{0x01, 0x01} // gender, success
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 SetAccountResult body: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=login/clientbound/SetAccountResult version=gms_v83 ida=0x5fc731
// packet-audit:verify packet=login/clientbound/SetAccountResult version=gms_v87 ida=0x634144
// packet-audit:verify packet=login/clientbound/SetAccountResult version=gms_v95 ida=0x5d5e80
// packet-audit:verify packet=login/clientbound/SetAccountResult version=gms_v84 ida=0x611809
// gms_v72: GENDER_DONE — CLogin::OnSetAccountResult @0x5b553a: Decode1(gender)@0x5b5553, Decode1(success)@0x5b5555; matches atlas SetAccountResult (gender+success bytes).
// packet-audit:verify packet=login/clientbound/SetAccountResult version=gms_v72 ida=0x5b553a
// packet-audit:verify packet=login/clientbound/SetAccountResult version=gms_v79 ida=0x5d07a2
func TestSetAccountResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetAccountResult{gender: 1, success: true}
			output := SetAccountResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Gender() != input.Gender() {
				t.Errorf("gender: got %v, want %v", output.Gender(), input.Gender())
			}
			if output.Success() != input.Success() {
				t.Errorf("success: got %v, want %v", output.Success(), input.Success())
			}
		})
	}
}
