package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61: LOGIN_STATUS temp-ban arm (resultCode v4==2) of
// CLogin::OnCheckPasswordResult @0x5657ce: head Decode1(result)@0x5657f9 +
// Decode1@0x5657ff + Decode4@0x56580d, then Decode1(reason)@0x56583f +
// DecodeBuffer(8)@0x565848 (ban-expiry FILETIME). Matches atlas AuthTemporaryBan
// GMS (bannedCode+0+int(0)+reason+long). bannedCode=2,reason=5,until=0.
//
// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v61 ida=0x5657ce
func TestAuthTemporaryBanV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := AuthTemporaryBan{bannedCode: 2, reason: 5, until: 0}
	want := []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0, 0, 0, 0, 0, 0, 0, 0}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 AuthTemporaryBan body: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v83 ida=0x5f83ee
// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v87 ida=0x62fb84
// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v95 ida=0x5dc600
// gms_v72: LOGIN_STATUS temp-ban path — CLogin::OnCheckPasswordResult @0x5b2577 (v104==2 branch: Decode1 reason + DecodeBuffer(8) ban-expiry date); legacy shape == v79.
// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v72 ida=0x5b2577
// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v79 ida=0x5cd38f
// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v84 ida=0x60d368
func TestAuthTemporaryBanRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AuthTemporaryBan{bannedCode: 2, reason: 3, until: 116444736000000000}
			output := AuthTemporaryBan{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.BannedCode() != input.BannedCode() {
				t.Errorf("bannedCode: got %v, want %v", output.BannedCode(), input.BannedCode())
			}
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
			if output.Until() != input.Until() {
				t.Errorf("until: got %v, want %v", output.Until(), input.Until())
			}
		})
	}
}
