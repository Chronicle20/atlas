package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v83 ida=0x66c23b
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v87 ida=0x6a7106
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v95 ida=0x640c50
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=jms_v185 ida=0x6e99c8
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v84 ida=0x68253d
func TestMonsterMovementAck(t *testing.T) {
	input := NewMonsterMovementAck(5001, 42, 300, true, 10, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
