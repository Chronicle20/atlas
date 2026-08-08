package clientbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// u16 reinterprets a signed 16-bit field as its little-endian wire value.
// A direct uint16(int16(-300)) is a constant expression and will not compile.
func u16(v int16) uint16 { return uint16(v) }

// TestNpcSpawnByteOutputV48 pins the gms_v48 NPC_ENTER_FIELD clientbound wire.
//
// IDA (GMS_v48_1_DEVM.exe): CNpcPool::OnNpcEnterField @0x56d527 reads
// Decode4(objectId) @0x56d539, then — on the not-already-spawned branch —
// Decode4(templateId) @0x56d570, and delegates the remainder to CNpc::Init
// (sub_566A30 @0x566a30). That body reads exactly six fields:
//
//	Decode2 @0x566a64 → x
//	Decode2 @0x566a72 → cy
//	Decode1 @0x566a9a → f
//	Decode2 @0x566aaa → fh
//	Decode2 @0x566aca → rx0
//	Decode2 @0x566ad8 → rx1
//
// and then makes no further CInPacket call — the rest of the function is
// layer/canvas setup. Eight reads total: 4+4+2+2+1+2+2+2 = 19 bytes.
//
// v61 CNpc::Init @0x5e7bef, v72 @0x63d7c1 and v79 @0x660154 read those same six
// plus a trailing Decode1 (bEnabled), matching v83 @0x6d9993 / v87 @0x716fd5 /
// v95 @0x679680 at nine reads. Atlas wrote that byte unconditionally, so v48
// received 20 bytes for a 19-byte packet.
//
// packet-audit:verify packet=npc/clientbound/NpcSpawn version=gms_v48 ida=0x56d527
func TestNpcSpawnByteOutputV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewNpcSpawn(100, 9010000, 150, -300, 0, 500, -50, 250)

	got := in.Encode(l, pt.CreateContext("GMS", 48, 1))(nil)

	want := make([]byte, 0, 19)
	want = binary.LittleEndian.AppendUint32(want, 100)     // objectId
	want = binary.LittleEndian.AppendUint32(want, 9010000) // templateId
	want = binary.LittleEndian.AppendUint16(want, uint16(150))
	want = binary.LittleEndian.AppendUint16(want, u16(-300))
	want = append(want, 0x01) // f == 0 encodes as 1
	want = binary.LittleEndian.AppendUint16(want, 500)
	want = binary.LittleEndian.AppendUint16(want, u16(-50))
	want = binary.LittleEndian.AppendUint16(want, 250)

	if !bytes.Equal(got, want) {
		t.Errorf("v48 npc spawn:\n got %v (%d bytes)\nwant %v (%d bytes)", got, len(got), want, len(want))
	}
}

// TestNpcSpawnEnabledFlagBoundary guards the v48/v61 boundary directly: v48 must
// stop after rx1, every later GMS version and JMS must carry the trailing byte.
func TestNpcSpawnEnabledFlagBoundary(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewNpcSpawn(100, 9010000, 150, -300, 0, 500, -50, 250)

	if n := len(in.Encode(l, pt.CreateContext("GMS", 48, 1))(nil)); n != 19 {
		t.Errorf("GMS v48 length = %d, want 19 (no trailing bEnabled)", n)
	}
	for _, v := range []uint16{61, 72, 79, 83, 87, 95} {
		b := in.Encode(l, pt.CreateContext("GMS", v, 1))(nil)
		if len(b) != 20 {
			t.Errorf("GMS v%d length = %d, want 20 (trailing bEnabled)", v, len(b))
			continue
		}
		if b[19] != 0x01 {
			t.Errorf("GMS v%d trailing byte = 0x%02x, want 0x01", v, b[19])
		}
	}
	if n := len(in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)); n != 20 {
		t.Errorf("JMS v185 length = %d, want 20 (trailing bEnabled)", n)
	}
}

// TestNpcSpawnRequestControllerV48Boundary pins the same v48/v61 boundary for
// CNpcPool::OnNpcChangeController. IDA (GMS_v48_1_DEVM.exe): @0x56d617 reads
// Decode1(flag) then Decode4(id), and the local-npc arm CNpcPool::SetLocalNpc
// @0x56d267 reads Decode4(template) before calling sub_566A30 @0x566a30 - the
// same CNpc::Init NpcSpawn delegates to, which stops after rx1. Nine reads on
// v48 (20 bytes with the leading flag), ten on v83 @0x6d9a83 / v87 @0x7170c8 /
// v95 @0x679730.
//
// packet-audit:verify packet=npc/clientbound/NpcSpawnRequestController version=gms_v48 ida=0x56d617
func TestNpcSpawnRequestControllerV48Boundary(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewNpcSpawnRequestController(1, 100, 9010000, 150, -300, 0, 500, -50, 250, true)

	got := in.Encode(l, pt.CreateContext("GMS", 48, 1))(nil)

	want := []byte{0x01}
	want = binary.LittleEndian.AppendUint32(want, 100)
	want = binary.LittleEndian.AppendUint32(want, 9010000)
	want = binary.LittleEndian.AppendUint16(want, uint16(150))
	want = binary.LittleEndian.AppendUint16(want, u16(-300))
	want = append(want, 0x01)
	want = binary.LittleEndian.AppendUint16(want, 500)
	want = binary.LittleEndian.AppendUint16(want, u16(-50))
	want = binary.LittleEndian.AppendUint16(want, 250)

	if !bytes.Equal(got, want) {
		t.Errorf("v48 npc change-controller:\n got %v (%d bytes)\nwant %v (%d bytes)", got, len(got), want, len(want))
	}

	for _, v := range []uint16{61, 72, 79, 83, 87, 95} {
		b := in.Encode(l, pt.CreateContext("GMS", v, 1))(nil)
		if len(b) != len(want)+1 {
			t.Errorf("GMS v%d length = %d, want %d (trailing miniMap)", v, len(b), len(want)+1)
			continue
		}
		if b[len(want)] != 0x01 {
			t.Errorf("GMS v%d trailing byte = 0x%02x, want 0x01", v, b[len(want)])
		}
	}
}
