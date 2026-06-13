package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v83 ida=0x5f92ae
// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v87 ida=0x630af9
// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v95 ida=0x5d2250
func TestServerStatusRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerStatus{status: 1}
			output := ServerStatus{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Status() != input.Status() {
				t.Errorf("status: got %v, want %v", output.Status(), input.Status())
			}
		})
	}
}
