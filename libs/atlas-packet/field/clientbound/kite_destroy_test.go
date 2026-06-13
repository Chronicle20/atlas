package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldKiteDestroy version=gms_v83 ida=0x65b2ca
// packet-audit:verify packet=field/clientbound/FieldKiteDestroy version=gms_v87 ida=0x69544f
// packet-audit:verify packet=field/clientbound/FieldKiteDestroy version=gms_v95 ida=0x635d60
// packet-audit:verify packet=field/clientbound/FieldKiteDestroy version=jms_v185 ida=0x6d5f7f
func TestKiteDestroy(t *testing.T) {
	input := NewKiteDestroy(1, KiteDestroyAnimationType2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
