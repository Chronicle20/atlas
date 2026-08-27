package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
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

// TestNpcActionByteOutputV95 pins the gms_v95 NPC_ACTION (opcode 0x13A/314)
// clientbound animation-arm wire. IDA-verified client decode (database
// ecc757f4): the pool dispatcher CNpcPool::OnNpcPacket@0x679260 does
// Decode4(npcId) → CNpcPool::GetNpc, then switch(nType), case 314 routes to
// CNpc::OnMove@0x678060:
//
//	v4 = CInPacket::Decode1(iPacket)   @0x678099 -> action  (atlas unk).
//	v5 = CInPacket::Decode1(v3)        @0x6780a1 -> nChatIdx (atlas unk2).
//	... if (this->m_pTemplate->bMove) CMovePath::OnMovePacket(&v41[145], v3, 0)
//	    @0x6785df — the movement body is gated on the npc template's static
//	    bMove flag, not on any packet field (Task 11: CMovePath::OnMovePacket
//	    @0x6683f0 is the single shared v95 movepath decode for all four
//	    OnMove handlers). The animation arm carries no movepath, so the wire
//	    is exactly int(objectId — dispatcher prefix) + byte(unk) + byte(unk2)
//	    = 6 bytes, byte-identical to the verified v72/v79 read order.
//
// packet-audit:verify packet=npc/clientbound/NpcAction version=gms_v95 ida=0x678060
func TestNpcActionByteOutputV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext(pt.Variants[3].Region, pt.Variants[3].MajorVersion, pt.Variants[3].MinorVersion)
	// objectId=0x01020304, unk=2, unk2=1.
	input := NewNpcActionAnimation(0x01020304, 2, 1)
	expected := []byte{
		0x04, 0x03, 0x02, 0x01, // objectId (uint32-LE)
		0x02, // unk
		0x01, // unk2
	}
	if got := input.Encode(l, ctx)(nil); !bytes.Equal(got, expected) {
		t.Errorf("v95 npc-action golden mismatch: got %v want %v", got, expected)
	}
}
