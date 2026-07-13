package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestNpcActionByteOutputV79 pins the gms_v79 NPC_ACTION (op 0x0EE) clientbound
// animation-arm wire. IDA-verified client decode (GMS_v79_1_DEVM.exe, port
// 13340): the objectId is consumed by the pool dispatcher
// CNpcPool::OnNpcPacket @0x668999 (CInPacket::Decode4 → GetNpc), which routes
// case 238 to CNpc::OnMove @0x662203:
//
//	v3 = CInPacket::Decode1(a2)   @0x662220 → unk  (action).
//	v4 = CInPacket::Decode1(a2)   @0x662223 → unk2 (chatIdx).
//	... CMovePath::OnMovePacket    @0x66264e (only when the npc carries a movepath).
//
// The animation form carries no movepath, so the wire is exactly
// int(objectId) + byte(unk) + byte(unk2) = 6 bytes.
//
// packet-audit:verify packet=npc/clientbound/NpcAction version=gms_v79 ida=0x662203
func TestNpcActionByteOutputV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	// objectId=0x01020304, unk=2, unk2=1.
	input := NewNpcActionAnimation(0x01020304, 2, 1)
	expected := []byte{
		0x04, 0x03, 0x02, 0x01, // objectId (uint32-LE)
		0x02, // unk
		0x01, // unk2
	}
	if got := input.Encode(l, ctx)(nil); !bytes.Equal(got, expected) {
		t.Errorf("v79 npc-action golden mismatch: got %v want %v", got, expected)
	}
}
