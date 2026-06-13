package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v83 ida=0x5f83ee
// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v87 ida=0x62fb84
// packet-audit:verify packet=login/clientbound/AuthLoginFailed version=gms_v95 ida=0x5dc600
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
