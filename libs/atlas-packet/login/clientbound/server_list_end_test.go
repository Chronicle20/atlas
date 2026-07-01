package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v83 ida=0x5f95b7
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v87 ida=0x630e7c
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v95 ida=0x5da7f0
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v84 ida=0x60e5b3
// packet-audit:verify packet=login/clientbound/ServerListEnd version=jms_v185 ida=0x66f107
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v79 ida=0x5ce269
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v72 ida=0x5b33f8
func TestServerListEndRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerListEnd{}
			output := ServerListEnd{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
