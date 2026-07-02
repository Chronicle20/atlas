package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61: CLogin::OnCheckPasswordResult @0x5657ce (GMS_v61.1_U_DEVM.exe, port
// 13338) reads the LOGIN_STATUS body head as Decode1(status)@0x5657f9 +
// Decode1@0x5657ff + Decode4@0x56580d — byte + byte + int, matching the encoder's
// WriteByte(reason)+WriteByte(0)+(GMS)WriteInt(0). reason=5 → 05 00 00000000.
//
// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v61 ida=0x5657ce
func TestAuthLoginFailedV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := AuthLoginFailed{reason: 5}
	want := []byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00} // reason, 0, GMS int(0)
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 AuthLoginFailed body: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v83 ida=0x5f83ee
// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v84 ida=0x60d368
// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v87 ida=0x62fb84
// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v95 ida=0x5dc600
// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v79 ida=0x5cd38f
// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v72 ida=0x5b2577
func TestAuthLoginFailedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AuthLoginFailed{reason: 5}
			output := AuthLoginFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
		})
	}
}
