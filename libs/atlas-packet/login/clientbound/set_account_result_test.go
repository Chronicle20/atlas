package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
