package serverbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
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
// packet-audit:verify packet=npc/serverbound/NpcActionRequest version=gms_v84 ida=0x6ea340
//
// jms_v185 is verified by TestNPCActionBytesJMS185 below, against captured
// wire. A RoundTrip cannot catch a tail the client sends and the decoder
// never reads.
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

// TestNPCActionBytesJMS185 pins the live-captured jms_v185 NPC_ACTION wire
// frame `[PKT IN] handler=NPCActionHandle op=0x00d0 len=40`, captured on the
// atlas-main k3s environment (same session as the character-move captures).
// Prior to the npcMoveKeyPadTail fix Atlas decoded objectId/unk/unk2 and the
// one-element movement correctly but silently dropped the trailing 9-byte
// keypad-count(0) + bounding-rect tail. See npcMoveKeyPadTail and
// docs/tasks/fix-jms185-attack-decode/sibling-movement-ops-findings.md §1.
//
// packet-audit:verify packet=npc/serverbound/NpcActionRequest version=jms_v185 ida=0x7199ce
func TestNPCActionBytesJMS185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("JMS", 185, 1)

	// Frame body with the 2-byte opcode (0xd0 0x00) stripped, exactly as the
	// handler's reader sees it.
	bodyHex := "02000000ffff41037d00010041037d00000000000900000000000538040041037d0041037d00"
	body, err := hex.DecodeString(bodyHex)
	if err != nil {
		t.Fatalf("bad fixture hex: %v", err)
	}

	opts := test.MovementTypesJMS185()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m ActionRequest
	m.Decode(l, ctx)(&reader, opts)

	// The frame must now be consumed to the last byte, including the
	// keypad + bounding-rect tail (npcMoveKeyPadTail). Prior to the fix this
	// left 9 unconsumed bytes.
	if reader.Available() != 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", reader.Available())
	}
	if m.ObjectId() != 2 {
		t.Errorf("expected objectId 2, got %d", m.ObjectId())
	}
	if m.Unk() != 0xff || m.Unk2() != 0xff {
		t.Errorf("expected unk/unk2 0xff/0xff, got %#x/%#x", m.Unk(), m.Unk2())
	}
	if !m.HasMovement() {
		t.Fatal("expected hasMovement true")
	}
	mv := m.MovementData()
	if mv.StartX != 833 || mv.StartY != 125 {
		t.Errorf("expected origin 833,125, got %d,%d", mv.StartX, mv.StartY)
	}
	if len(mv.Elements) != 1 {
		t.Errorf("expected 1 movement element, got %d", len(mv.Elements))
	}
	if got := len(m.KeyPadStates()); got != 0 {
		t.Errorf("expected 0 keypad entries, got %d", got)
	}
	gl, gt, gr, gb := m.MoveRect()
	if gl != 833 || gt != 125 || gr != 833 || gb != 125 {
		t.Errorf("expected move rect 833,125,833,125, got %d,%d,%d,%d", gl, gt, gr, gb)
	}

	if got := test.Encode(t, ctx, m.Encode, opts); !bytes.Equal(got, body) {
		t.Fatalf("jms185 npc-action re-encode:\n got=% X\nwant=% X", got, body)
	}
}
