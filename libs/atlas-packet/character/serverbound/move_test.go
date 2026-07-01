package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestCharacterMoveByteV79 pins the gms_v79 MOVE_PLAYER (op 0x27) serverbound wire.
//
// IDA: CVecCtrlUser::EndUpdateActive @0x91b6e6 (renamed from sub_91B6E6;
// GMS_v79_1_DEVM.exe, port 13340) builds COutPacket(39):
//
//	Encode1 fieldKey  (*(get_field()+276))  @0x91b89f
//	Encode4 crc       (*(get_field()+483))  @0x91b8b2
//	CMovePath::Flush(&pkt) movement blob                @0x91b8c0
//
// v79 major 79 < 84 so the dr0/dr1/dr2/dr3/dwKey/crc32 anti-cheat header (added
// at GMS v84) is ABSENT — the wire is the v83-style lean layout fieldKey+crc.
// The movement blob is written by CMovePath::Flush; its bytes are OPAQUE (§5
// OPAQUE-EXCEPTION: the export's calls stop at the Flush boundary) and are
// derived here from the Atlas model.Movement encoder (StartX Int16 + StartY
// Int16 + count byte), not from a per-field decompile line.
//
// packet-audit:verify packet=character/serverbound/Move version=gms_v79 ida=0x91b6e6
func TestCharacterMoveByteV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	p := Move{fieldKey: 0x2A, crc: 500, movement: model.Movement{StartX: 10, StartY: 20}}
	got := p.Encode(l, ctx)(nil)
	want := []byte{
		0x2A,                   // fieldKey        @0x91b89f
		0xF4, 0x01, 0x00, 0x00, // crc=500         @0x91b8b2
		0x0A, 0x00, // movement StartX=10  (opaque, CMovePath::Flush @0x91b8c0)
		0x14, 0x00, // movement StartY=20  (opaque)
		0x00, // movement element count=0 (opaque)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 Move: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/Move version=gms_v87 ida=0xa5c937
// packet-audit:verify packet=character/serverbound/Move version=gms_v95 ida=0x9a0d20
// packet-audit:verify packet=character/serverbound/Move version=jms_v185 ida=0xaaa076
// packet-audit:verify packet=character/serverbound/Move version=gms_v84 ida=0xa1334e
func TestCharacterMove(t *testing.T) {
	p := Move{}
	p.dr0 = 100
	p.dr1 = 200
	p.fieldKey = 42
	p.dr2 = 300
	p.dr3 = 400
	p.crc = 500
	p.dwKey = 600
	p.crc32 = 700

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.FieldKey() != 42 {
				t.Errorf("expected fieldKey 42, got %d", p.FieldKey())
			}
			// dr0/dr1/dr2/dr3/dwKey/crc32 are CONFIRMED v84+ against the v84 client
			// self-move senders (sub_A1334E / sub_9843EA write the full dr-block;
			// v83 writes only fieldKey+crc). JMS has no dr fields (v83-equivalent).
			if v.Region == "GMS" && v.MajorVersion >= 84 {
				if p.Dr0() != 100 {
					t.Errorf("expected dr0 100, got %d", p.Dr0())
				}
				if p.Dr1() != 200 {
					t.Errorf("expected dr1 200, got %d", p.Dr1())
				}
				if p.Dr2() != 300 {
					t.Errorf("expected dr2 300, got %d", p.Dr2())
				}
				if p.Dr3() != 400 {
					t.Errorf("expected dr3 400, got %d", p.Dr3())
				}
				if p.DwKey() != 600 {
					t.Errorf("expected dwKey 600, got %d", p.DwKey())
				}
				if p.Crc32() != 700 {
					t.Errorf("expected crc32 700, got %d", p.Crc32())
				}
			}
			if v.Region == "GMS" && v.MajorVersion > 28 && p.Crc() != 500 {
				t.Errorf("expected crc 500, got %d", p.Crc())
			}
		})
	}
}

func TestCharacterMoveOperationString(t *testing.T) {
	p := Move{}
	if p.Operation() != CharacterMoveHandle {
		t.Errorf("expected operation %s, got %s", CharacterMoveHandle, p.Operation())
	}
	if p.String() == "" {
		t.Error("expected non-empty string")
	}
}
