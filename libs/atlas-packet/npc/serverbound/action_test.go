package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestNPCActionByteOutputV79 pins the gms_v79 NPC_ACTION (op 0x0BD) serverbound
// wire. IDA-verified send site (GMS_v79_1_DEVM.exe, port 13340) —
// CNpc::GenerateMovePath @0x66266f, send block:
//
//	COutPacket::COutPacket(189)              @0x66270e → opcode 0xBD (registry).
//	COutPacket::Encode4(this+41 = npcOid)    @0x662721 → objectId uint32-LE.
//	COutPacket::Encode1(pExceptionObject)    @0x66272c → unk  byte.
//	COutPacket::Encode1(a3)                  @0x662737 → unk2 byte.
//	if (npc has movepath) CMovePath::Flush   @0x662753 → trailing movepath.
//
// The no-movement form omits the movepath, so the wire is exactly
// int(objectId) + byte(unk) + byte(unk2) = 6 bytes.
//
// packet-audit:verify packet=npc/serverbound/NpcActionRequest version=gms_v79 ida=0x66266f
func TestNPCActionByteOutputV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	input := ActionRequest{objectId: 0x01020304, unk: 1, unk2: 2}
	expected := []byte{
		0x04, 0x03, 0x02, 0x01, // objectId (uint32-LE)
		0x01, // unk
		0x02, // unk2
	}
	if got := input.Encode(l, ctx)(nil); !bytes.Equal(got, expected) {
		t.Errorf("v79 npc-action-request golden mismatch: got %v want %v", got, expected)
	}
}

// packet-audit:verify packet=npc/serverbound/NpcActionRequest version=gms_v95 ida=0x671590
// packet-audit:verify packet=npc/serverbound/NpcActionRequest version=jms_v185 ida=0x7199ce
// packet-audit:verify packet=npc/serverbound/NpcActionRequest version=gms_v84 ida=0x6ea340
func TestNPCActionWithoutMovement(t *testing.T) {
	p := ActionRequest{}
	p.objectId = 12345
	p.unk = 1
	p.unk2 = 2
	p.hasMovement = false

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.ObjectId() != 12345 {
				t.Errorf("expected objectId 12345, got %d", p.ObjectId())
			}
			if p.Unk() != 1 {
				t.Errorf("expected unk 1, got %d", p.Unk())
			}
			if p.Unk2() != 2 {
				t.Errorf("expected unk2 2, got %d", p.Unk2())
			}
			if p.HasMovement() {
				t.Error("expected hasMovement false")
			}
		})
	}
}

func TestNPCActionWithMovement(t *testing.T) {
	p := ActionRequest{}
	p.objectId = 99999
	p.unk = 3
	p.unk2 = 4
	p.hasMovement = true
	// movement with 0 elements (startX=10, startY=20)
	p.movement.StartX = 10
	p.movement.StartY = 20

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.ObjectId() != 99999 {
				t.Errorf("expected objectId 99999, got %d", p.ObjectId())
			}
			if !p.HasMovement() {
				t.Error("expected hasMovement true")
			}
			if p.MovementData().StartX != 10 {
				t.Errorf("expected startX 10, got %d", p.MovementData().StartX)
			}
			if p.MovementData().StartY != 20 {
				t.Errorf("expected startY 20, got %d", p.MovementData().StartY)
			}
		})
	}
}

func TestNPCActionOperationString(t *testing.T) {
	p := ActionRequest{}
	if p.Operation() != NPCActionHandle {
		t.Errorf("expected operation %s, got %s", NPCActionHandle, p.Operation())
	}
	if p.String() == "" {
		t.Error("expected non-empty string")
	}
}
