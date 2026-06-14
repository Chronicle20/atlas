package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v83 ida=0x5f83ee
// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v87 ida=0x62fb84
// packet-audit:verify packet=login/clientbound/AuthTemporaryBan version=gms_v95 ida=0x5dc600
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
