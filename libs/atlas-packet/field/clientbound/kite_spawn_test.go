package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldKiteSpawn version=gms_v83 ida=0x65acdf
// packet-audit:verify packet=field/clientbound/FieldKiteSpawn version=gms_v87 ida=0x694e48
// packet-audit:verify packet=field/clientbound/FieldKiteSpawn version=gms_v95 ida=0x6369c0
// packet-audit:verify packet=field/clientbound/FieldKiteSpawn version=jms_v185 ida=0x6d5978
// packet-audit:verify packet=field/clientbound/FieldKiteSpawn version=gms_v84 ida=0x670ac0
func TestKiteSpawn(t *testing.T) {
	input := NewKiteSpawn(1, 5010000, "Hello World!", "Player1", 100, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
