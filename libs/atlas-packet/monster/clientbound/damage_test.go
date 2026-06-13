package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterDamage version=gms_v83 ida=0x66c6c2
// packet-audit:verify packet=monster/clientbound/MonsterDamage version=gms_v87 ida=0x6a758d
// packet-audit:verify packet=monster/clientbound/MonsterDamage version=gms_v95 ida=0x64ecb0
// packet-audit:verify packet=monster/clientbound/MonsterDamage version=jms_v185 ida=0x6e9e43
func TestMonsterDamage(t *testing.T) {
	input := NewMonsterDamage(5001, MonsterDamageTypeUnk2, 1500, 8500, 10000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
