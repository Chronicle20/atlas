package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNpcActionByteOutputV72 pins the gms_v72 NPC_ACTION (op 230) clientbound
// animation-arm wire. IDA-verified client decode (GMS_v72.1_U_DEVM.exe, port
// 13339): the objectId is consumed by the pool dispatcher
// CNpcPool::OnNpcPacket @0x645dfe (CInPacket::Decode4 -> GetNpc), which routes
// case 230 to CNpc::OnMove @0x63f831:
//
//	v3 = CInPacket::Decode1(a2)   -> unk  (action).
//	v4 = CInPacket::Decode1(a2)   -> unk2 (chatIdx).
//	... CMovePath::OnMovePacket    @0x635bc2 (only when the npc carries a movepath).
//
// Byte-identical to the verified v79 read order. The animation form carries no
// movepath, so the wire is exactly int(objectId) + byte(unk) + byte(unk2) = 6
// bytes.
//
// packet-audit:verify packet=npc/clientbound/NpcAction version=gms_v72 ida=0x63f831
func TestNpcActionByteOutputV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewNpcActionAnimation(0x01020304, 2, 1)
	expected := []byte{
		0x04, 0x03, 0x02, 0x01, // objectId (uint32-LE)
		0x02, // unk
		0x01, // unk2
	}
	if got := input.Encode(l, ctx)(nil); !bytes.Equal(got, expected) {
		t.Errorf("v72 npc-action golden mismatch: got %v want %v", got, expected)
	}
}
