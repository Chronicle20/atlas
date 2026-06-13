package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v83 ida=0x937607
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v87 ida=0x9b1d1e
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v95 ida=0x8ed310
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=jms_v185 ida=0x9f6199
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v84 ida=0x96e8c0
func TestChalkboardUse(t *testing.T) {
	input := NewChalkboardUse(1234, "Selling scrolls!")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestChalkboardClear(t *testing.T) {
	input := NewChalkboardClear(1234)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
