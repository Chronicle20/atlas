package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v83 ida=0x5f83ee
// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v87 ida=0x62fb84
// packet-audit:verify packet=login/clientbound/AuthPermanentBan version=gms_v95 ida=0x5dc600
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
