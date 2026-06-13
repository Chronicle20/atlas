package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=portal/serverbound/PortalScript version=gms_v83 ida=0x94dac6
// packet-audit:verify packet=portal/serverbound/PortalScript version=gms_v87 ida=0x9c8832
// packet-audit:verify packet=portal/serverbound/PortalScript version=gms_v95 ida=0x919a10
// packet-audit:verify packet=portal/serverbound/PortalScript version=jms_v185 ida=0xa0dde7
func TestScriptRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Script{fieldKey: 1, portalName: "sp", x: 100, y: 200}
			output := Script{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PortalName() != input.PortalName() {
				t.Errorf("portalName: got %v, want %v", output.PortalName(), input.PortalName())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}
