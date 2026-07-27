package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61: LOGIN_STATUS perm-ban arm (resultCode v4==27) of
// CLogin::OnCheckPasswordResult @0x5657ce reads only the shared head (Decode1
// @0x5657f9 + Decode1@0x5657ff + Decode4@0x56580d) then routes to a dialog
// (CUtilDlg::YesNo @0x565c5a); no trailing reason/timestamp. Matches atlas
// AuthPermanentBan GMS path (bannedCode+0+int(0), trailing bytes skipped for GMS).
//
// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v61 ida=0x5657ce
func TestAuthPermanentBanV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := AuthPermanentBan{bannedCode: 27}
	want := []byte{0x1B, 0x00, 0x00, 0x00, 0x00, 0x00}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 AuthPermanentBan body: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v83 ida=0x5f83ee
// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v87 ida=0x62fb84
// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v95 ida=0x5dc600
// gms_v72: LOGIN_STATUS ban path — CLogin::OnCheckPasswordResult @0x5b2577 (v104==2 branch @0x5b25dd: Decode1 reason, DecodeBuffer(8) date); legacy shape == v79.
// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v72 ida=0x5b2577
// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v79 ida=0x5cd38f
// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v84 ida=0x60d368
func TestAuthPermanentBanRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AuthPermanentBan{bannedCode: 2}
			output := AuthPermanentBan{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.BannedCode() != input.BannedCode() {
				t.Errorf("bannedCode: got %v, want %v", output.BannedCode(), input.BannedCode())
			}
		})
	}
}
