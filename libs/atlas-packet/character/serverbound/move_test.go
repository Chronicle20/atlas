package serverbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
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

// TestCharacterMoveByteV61 pins the very-legacy GMS v61 MOVE_PLAYER (op 38) serverbound
// wire, which OMITS the 4-byte crc that v72+ carry.
//
// IDA: CVecCtrlUser::EndUpdateActive @0x801109 (GMS_v61.1_U_DEVM.exe, port 13338 — named
// from sub_801109) builds COutPacket(38):
//
//	Encode1 fieldKey  (*(get_field()+248))  @0x8012c3
//	CMovePath::Flush(&pkt) movement blob                @0x8012d1
//
// There is NO Encode4(crc) between fieldKey and Flush — unlike v72
// CVecCtrlUser::EndUpdateActive @0x8cb63e (fieldKey+crc+Flush). The move-crc was added
// after v61; the codec now gates it >=72 (was the incorrect >28 assumption). v61 < 84 so
// the dr-block header is also absent. The movement blob is written by CMovePath::Flush;
// its bytes are OPAQUE (§5 OPAQUE-EXCEPTION — the export's calls stop at the Flush
// boundary) and derive from the shared model.Movement encoder (StartX Int16 + StartY
// Int16 + count byte), not a per-field decompile line.
//
// packet-audit:verify packet=character/serverbound/Move version=gms_v61 ida=0x801109
func TestCharacterMoveByteV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	p := Move{fieldKey: 0x2A, crc: 500, movement: model.Movement{StartX: 10, StartY: 20}}
	got := p.Encode(l, ctx)(nil)
	want := []byte{
		0x2A, // fieldKey        @0x8012c3
		// NO crc (v61 < 72)
		0x0A, 0x00, // movement StartX=10  (opaque, CMovePath::Flush @0x8012d1)
		0x14, 0x00, // movement StartY=20  (opaque)
		0x00, // movement element count=0 (opaque)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 Move: got % x, want % x", got, want)
	}
}

// TestCharacterMoveByteV48 pins the very-legacy GMS v48 MOVE_PLAYER (op 33) serverbound
// wire, byte-identical to v61: OMITS the crc (v48 < 72) and the dr-block (v48 < 84).
//
// IDA: sub_6E9923 (GMS_v48_1_DEVM.exe, port 13337) builds COutPacket(33) @0x6e9ac1:
//
//	Encode1 fieldKey  (*(get_field()+216))  @0x6e9add
//	CMovePath::Flush(&pkt) (sub_5622DA) movement blob   @0x6e9aeb
//
// There is NO Encode4(crc) between fieldKey and Flush. The movement blob is written by
// CMovePath::Flush; bytes OPAQUE (§5) from the shared model.Movement encoder.
//
// packet-audit:verify packet=character/serverbound/Move version=gms_v48 ida=0x6e9923
func TestCharacterMoveByteV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	p := Move{fieldKey: 0x2A, crc: 500, movement: model.Movement{StartX: 10, StartY: 20}}
	got := p.Encode(l, ctx)(nil)
	want := []byte{
		0x2A, // fieldKey        @0x6e9add
		// NO crc (v48 < 72)
		0x0A, 0x00, // movement StartX=10  (opaque, CMovePath::Flush @0x6e9aeb)
		0x14, 0x00, // movement StartY=20  (opaque)
		0x00, // movement element count=0 (opaque)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 Move: got % x, want % x", got, want)
	}
}

// TestCharacterMoveByteV72 pins the gms_v72 MOVE_PLAYER (op 40) serverbound wire.
//
// IDA: CVecCtrlUser::EndUpdateActive @0x8cb63e (GMS_v72.1_U_DEVM.exe, port 13339)
// builds COutPacket(40):
//
//	Encode1 fieldKey  (*(get_field()+276))  @0x8cb7f7
//	Encode4 crc       (*(get_field()+476))  @0x8cb80a
//	CMovePath::Flush(&pkt) movement blob                @0x8cb818
//
// v72 major 72 < 84 so the dr0/dr1/dr2/dr3/dwKey/crc32 anti-cheat header (added at
// GMS v84) is ABSENT — byte-identical lean fieldKey+crc layout to the verified v79
// fixture. The movement blob is written by CMovePath::Flush; bytes OPAQUE (§5) and
// derived from the shared model.Movement encoder (v72 < 88 so no XOffset/YOffset),
// same as v79.
//
// packet-audit:verify packet=character/serverbound/Move version=gms_v72 ida=0x8cb63e
func TestCharacterMoveByteV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	p := Move{fieldKey: 0x2A, crc: 500, movement: model.Movement{StartX: 10, StartY: 20}}
	got := p.Encode(l, ctx)(nil)
	want := []byte{
		0x2A,                   // fieldKey        @0x8cb7f7
		0xF4, 0x01, 0x00, 0x00, // crc=500         @0x8cb80a
		0x0A, 0x00, // movement StartX=10  (opaque, CMovePath::Flush @0x8cb818)
		0x14, 0x00, // movement StartY=20  (opaque)
		0x00, // movement element count=0 (opaque)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 Move: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/Move version=gms_v87 ida=0xa5c937
// packet-audit:verify packet=character/serverbound/Move version=gms_v95 ida=0x9a0d20
// packet-audit:verify packet=character/serverbound/Move version=gms_v84 ida=0xa1334e
//
// jms_v185 is verified by TestCharacterMoveBytesJMS185 below, against captured
// wire. A RoundTrip cannot catch a header the client sends and the decoder
// never reads.
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
			// v83 writes only fieldKey+crc). jms v185 carries them too, from
			// captured wire — see moveDrBlocks.
			if (v.Region == "GMS" && v.MajorVersion >= 84) || v.Region == "JMS" {
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

// --- JMS v185.1 MOVE_PLAYER (live-capture derived) ---
//
// The jms sender for this op is the registry-primary CUserLocal::OnKey, which
// the retail SCY dump does not decompile (same code-flow virtualization that
// hides the attack senders' encode tails). CVecCtrlUser::EndUpdateActive
// @0xaaa076 DOES decompile and writes no dr words, which is why an earlier
// pass concluded jms used the GMS v83-style layout — but that function is a
// different send path and cannot produce any observed frame (see moveDrBlocks).
//
// These fixtures are pinned to `[PKT IN ]` frames captured from the live
// client on the atlas-main k3s environment (tenant
// abedf3b4-1d7c-4b3b-bc52-70f62ab09418, JMS 185.1, op=0x0020 / MOVE_PLAYER
// opcode 32, 2026-09-02). Nine frames were captured; the three below span the
// observed range. In all nine the CMovePath blob begins at frame offset 0x1f —
// behind the 29-byte dr header — and yields an origin and an element count
// that tracks the frame length. Derivation:
// docs/tasks/fix-jms185-attack-decode/diagnosis.md.

// packet-audit:verify packet=character/serverbound/Move version=jms_v185 ida=0xaaa076
func TestCharacterMoveBytesJMS185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("JMS", 185, 1)
	opts := test.MovementTypesJMS185()

	// Frame bodies with the 2-byte opcode stripped, exactly as the handler's
	// reader sees them.
	cases := []struct {
		name          string
		body          string
		originX       int16
		originY       int16
		elements      int
		keyPadEntries int
		rect          [4]int16
	}{
		// len 90 on the wire.
		{"two_elements", "e1ffffffc3795d8d00c33f6581ff17e5ff6fa2eb4b090000000ff0c922" +
			"b3ffd7000200b3ffd7000000000001000000000004a40100b9ffd7007d000000010000000000025a0011000000000000004404b3ffd700b9ffd700",
			-77, 215, 2, 17, [4]int16{-77, 215, -71, 215}},
		// len 108 on the wire — the frame quoted in the diagnosis.
		{"three_elements", "e1ffffffc3795d8d01ebbf6581131fe5ff198f2c641a000000a60d9b63" +
			"a3ffb8010300a3ffce0100002c0100000000000006960000a3ffd5010000000000000000000006140000a3ffd5010000000031000000000004540111000000000000000000a3ffb801a3ffd501",
			-93, 440, 3, 17, [4]int16{-93, 440, -93, 469}},
		// len 188 on the wire — the longest captured.
		{"eight_elements", "e1ffffffc3795d8d00c33f6581ff17e5ff6fa2eb4b090000000ff0c922" +
			"e700ce000800e700d70000000000000000000000060f0000e700d600000000000a0000000000026900016400d5fd06000000e700c600000011fe000000000000061e0000e700ac00080089fe000000000000063c0000e700a1000000c5fe000000000000061e0000e80088000800f1ff00000000000006960000e90096000900e10000000000000006780011444444444444444404e7008800e900d700",
			231, 206, 8, 17, [4]int16{231, 136, 233, 215}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, err := hex.DecodeString(c.body)
			if err != nil {
				t.Fatalf("bad fixture hex: %v", err)
			}
			req := request.Request(body)
			reader := request.NewRequestReader(&req, 0)
			var m Move
			m.Decode(l, ctx)(&reader, opts)

			// The frame must now be consumed to the last byte, including the
			// keypad + bounding-rect tail (moveKeyPadTail).
			if reader.Available() != 0 {
				t.Fatalf("reader has %d unconsumed bytes after decode", reader.Available())
			}
			if got := len(m.KeyPadStates()); got != c.keyPadEntries {
				t.Errorf("keypad entries = %d, want %d", got, c.keyPadEntries)
			}
			gl, gt, gr, gb := m.MoveRect()
			if gl != c.rect[0] || gt != c.rect[1] || gr != c.rect[2] || gb != c.rect[3] {
				t.Errorf("move rect = %d,%d,%d,%d, want %v", gl, gt, gr, gb, c.rect)
			}
			mv := m.MovementData()
			if len(mv.Elements) != c.elements {
				t.Errorf("elements = %d, want %d", len(mv.Elements), c.elements)
			}
			if mv.StartX != c.originX || mv.StartY != c.originY {
				t.Errorf("origin = %d,%d, want %d,%d", mv.StartX, mv.StartY, c.originX, c.originY)
			}

			if got := test.Encode(t, ctx, m.Encode, opts); !bytes.Equal(got, body) {
				t.Fatalf("jms185 move re-encode:\n got=% X\nwant=% X", got, body)
			}
		})
	}
}
