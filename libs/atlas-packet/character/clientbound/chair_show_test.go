package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v83 ida=0x9724f9
// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v84 ida=0x9b2518
// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v87 ida=0x9f74de
// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v95 ida=0x949240
// packet-audit:verify packet=character/clientbound/CharacterChairShow version=jms_v185 ida=0xa44324
//
// jms SHOW_CHAIR (opcode 0xCA) is read inline in CUserPool::OnUserRemotePacket
// case 0xCA@0xa44324: *(RemoteUser+16516) = Decode4(chairId), with characterId
// read by the dispatcher's leading Decode4 before the case — same wire shape as
// the v83 (case 0xC4) / v87 (case 0xD1) / v95 inline twins (characterId + chairId,
// two LE uint32s). The codec is version-agnostic.
func TestCharacterChairShow(t *testing.T) {
	input := NewCharacterChairShow(1234, 3010000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
