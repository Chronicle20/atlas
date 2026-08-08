package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 reactor byte fixtures. Opcodes come from CReactorPool::OnPacket @0x5a5390,
// which routes 210 -> OnReactorChangeState, 212 -> OnReactorEnterField and
// 213 -> OnReactorLeaveField (v61 uses 214/216/217).

// TestReactorSpawnBytesV48 pins the v48 reactor-spawn wire, which is the v83 wire
// MINUS the trailing name string.
//
// IDA CReactorPool::OnReactorEnterField @0x5a54b4 reads exactly six fields:
// Decode4 @0x5a54ec (id), Decode4 @0x5a54f8 (templateId), Decode1 @0x5a5503
// (state), Decode2 @0x5a5524 (x), Decode2 @0x5a5534 (y), Decode1 @0x5a5544
// (flags) - and then makes no further CInPacket call; the rest of the function
// is LoadReactorLayer/canvas setup. v72 @0x69207c and v79 @0x6b77bb read the same
// six. Only v83 @0x735127, v84 @0x75271c, v87 @0x77af9c and v95 @0x6cf490 append
// a DecodeStr, so the name arrived at v83 and Atlas was sending it to all four
// legacy versions.
//
// packet-audit:verify packet=reactor/clientbound/ReactorSpawn version=gms_v48 ida=0x5a54b4
func TestReactorSpawnBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewReactorSpawn(0x01020304, 2000, 1, 300, -400, 0, "gate")

	got := in.Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // id            — Decode4 @0x5a54ec
		0xD0, 0x07, 0x00, 0x00, // templateId    — Decode4 @0x5a54f8
		0x01,       // state         — Decode1 @0x5a5503
		0x2C, 0x01, // x = 300       — Decode2 @0x5a5524
		0x70, 0xFE, // y = -400      — Decode2 @0x5a5534
		0x00, // flags         — Decode1 @0x5a5544
		// no name string: absent until v83.
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 reactor spawn:\n got % x\nwant % x", got, want)
	}
}

// TestReactorSpawnNameBoundary guards the v79/v83 boundary: the legacy versions
// stop after the flags byte, v83 onward append the length-prefixed name.
func TestReactorSpawnNameBoundary(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewReactorSpawn(0x01020304, 2000, 1, 300, -400, 0, "gate")

	base := in.Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	for _, v := range []uint16{61, 72, 79} {
		if got := in.Encode(l, pt.CreateContext("GMS", v, 1))(nil); !bytes.Equal(got, base) {
			t.Errorf("GMS v%d should match v48 (no name):\n got % x\nwant % x", v, got, base)
		}
	}
	nameWire := []byte{0x04, 0x00, 'g', 'a', 't', 'e'}
	for _, v := range []uint16{83, 84, 87, 95} {
		got := in.Encode(l, pt.CreateContext("GMS", v, 1))(nil)
		want := append(append([]byte{}, base...), nameWire...)
		if !bytes.Equal(got, want) {
			t.Errorf("GMS v%d should carry the name:\n got % x\nwant % x", v, got, want)
		}
	}
}

// TestReactorHitBytesV48 — CReactorPool::OnReactorChangeState @0x5a53c4 reads
// Decode4(id), Decode1(state), Decode2(x), Decode2(y), Decode2(direction),
// Decode1, Decode1. Shape-stable; the existing codec already matches.
//
// packet-audit:verify packet=reactor/clientbound/ReactorHit version=gms_v48 ida=0x5a53c4
func TestReactorHitBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewReactorHit(0x01020304, 1, 300, -400, 2).Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // id        — Decode4
		0x01,       // state     — Decode1
		0x2C, 0x01, // x = 300   — Decode2
		0x70, 0xFE, // y = -400  — Decode2
		0x02, 0x00, // direction — Decode2
		0x00, // unk1 (constructor pins 0) — Decode1
		0x05, // unk2 (constructor pins 5) — Decode1
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 reactor hit:\n got % x\nwant % x", got, want)
	}
}

// TestReactorDestroyBytesV48 — CReactorPool::OnReactorLeaveField @0x5a588b reads
// Decode4(id), Decode1(state), Decode2(x), Decode2(y). Shape-stable.
//
// packet-audit:verify packet=reactor/clientbound/ReactorDestroy version=gms_v48 ida=0x5a588b
func TestReactorDestroyBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewReactorDestroy(0x01020304, 1, 300, -400).Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // id       — Decode4
		0x01,       // state    — Decode1
		0x2C, 0x01, // x = 300  — Decode2
		0x70, 0xFE, // y = -400 — Decode2
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 reactor destroy:\n got % x\nwant % x", got, want)
	}
}
