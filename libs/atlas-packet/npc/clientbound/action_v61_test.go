package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestNpcActionByteOutputV61 pins the gms_v61 NPC_ACTION clientbound
// animation-arm wire. IDA-verified client decode (GMS_v61.1_U_DEVM.exe, port
// 13338): the objectId is consumed by the pool dispatcher CNpcPool::OnNpcPacket
// @0x5efd04 (Decode4 -> GetNpc), which routes the packet to CNpc::OnMove
// @0x5e9c66:
//
//	v3 = CInPacket::Decode1(a2)   -> unk  (action).
//	v4 = CInPacket::Decode1(a2)   -> unk2 (chatIdx).
//	... CMovePath::OnMovePacket    (only when the npc carries a movepath).
//
// Byte-identical to the verified v72 read order. The animation form carries no
// movepath, so the wire is exactly int(objectId) + byte(unk) + byte(unk2) = 6
// bytes.
//
// packet-audit:verify packet=npc/clientbound/NpcAction version=gms_v61 ida=0x5e9c66
func TestNpcActionByteOutputV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	input := NewNpcActionAnimation(0x01020304, 2, 1)
	expected := []byte{
		0x04, 0x03, 0x02, 0x01, // objectId (uint32-LE)
		0x02, // unk
		0x01, // unk2
	}
	if got := input.Encode(l, ctx)(nil); !bytes.Equal(got, expected) {
		t.Errorf("v61 npc-action golden mismatch: got %v want %v", got, expected)
	}
}
