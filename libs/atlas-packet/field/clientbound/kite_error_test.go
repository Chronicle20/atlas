package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldKiteError version=gms_v83 ida=0x65acb4
// packet-audit:verify packet=field/clientbound/FieldKiteError version=gms_v87 ida=0x694e1d
// packet-audit:verify packet=field/clientbound/FieldKiteError version=gms_v95 ida=0x636760
// packet-audit:verify packet=field/clientbound/FieldKiteError version=jms_v185 ida=0x6d594d
// packet-audit:verify packet=field/clientbound/FieldKiteError version=gms_v84 ida=0x670a95
func TestKiteError(t *testing.T) {
	input := NewKiteError()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
