package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterHealth version=gms_v83 ida=0x66d639
// packet-audit:verify packet=monster/clientbound/MonsterHealth version=gms_v87 ida=0x6a8505
// packet-audit:verify packet=monster/clientbound/MonsterHealth version=gms_v95 ida=0x642ef0
// packet-audit:verify packet=monster/clientbound/MonsterHealth version=jms_v185 ida=0x6eaddf
func TestMonsterHealth(t *testing.T) {
	input := NewMonsterHealth(5001, 85)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
