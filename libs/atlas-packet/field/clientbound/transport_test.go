package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldTransport version=gms_v83 ida=0x54dd08
// packet-audit:verify packet=field/clientbound/FieldTransport version=gms_v87 ida=0x577c21
// packet-audit:verify packet=field/clientbound/FieldTransport version=gms_v95 ida=0x54d5a0
// packet-audit:verify packet=field/clientbound/FieldTransport version=jms_v185 ida=0x58e280
func TestFieldTransport(t *testing.T) {
	input := NewFieldTransport(TransportStateMove1, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
