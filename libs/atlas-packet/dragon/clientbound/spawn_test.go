package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// dragonSpawnBody is the SPAWN_DRAGON wire per CDragon::OnCreated (GMS v95.0
// @0x50dc90): Decode4 x, Decode4 y, Decode1 moveAction, Decode2 <discarded>,
// Decode2 jobCode. The leading owner character id is consumed upstream by
// CUserPool::OnUserCommonPacket (@0x94cdb0) before the family dispatch.
//
//	int  ownerCharacterId = 4242
//	int  x                = 100    (FOUR bytes, not two)
//	int  y                = -200   (FOUR bytes, not two)
//	byte stance           = 3
//	short <discarded>     = 0      (client reads and throws away)
//	short jobId           = 2214
var dragonSpawnBody = []byte{
	0x92, 0x10, 0x00, 0x00,
	0x64, 0x00, 0x00, 0x00,
	0x38, 0xFF, 0xFF, 0xFF,
	0x03,
	0x00, 0x00,
	0xA6, 0x08,
}

// dragonSpawnBodyV83 is the GMS v83 wire: identical up through the discarded
// short, but CDragon::OnCreated (@0x4fe502) reads no jobId — the read
// sequence stops there. 11 bytes total (vs 13 on versions that carry jobId).
var dragonSpawnBodyV83 = []byte{
	0x92, 0x10, 0x00, 0x00,
	0x64, 0x00, 0x00, 0x00,
	0x38, 0xFF, 0xFF, 0xFF,
	0x03,
	0x00, 0x00,
}

// packet-audit:verify packet=dragon/clientbound/DragonSpawn version=gms_v95 ida=0x50dc90
func TestDragonSpawnBytes(t *testing.T) {
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, dragonSpawnBody) {
		t.Fatalf("spawn bytes = % X, want % X", got, dragonSpawnBody)
	}
}

// packet-audit:verify packet=dragon/clientbound/DragonSpawn version=gms_v83 ida=0x4fe502
func TestDragonSpawnBytesV83(t *testing.T) {
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, dragonSpawnBodyV83) {
		t.Fatalf("spawn bytes = % X, want % X", got, dragonSpawnBodyV83)
	}
}

// jobId is present on v84/v87/v92/v95/JMS185 (13 bytes) and absent on v83
// (11 bytes) — see spawnHasJobId in spawn.go for the IDA grounding. If any
// column ever diverges from this split, this table is where it shows up
// first.
func TestDragonSpawnBytesIdenticalAcrossVersions(t *testing.T) {
	versionsWithJobId := []struct {
		region string
		major  uint16
	}{
		{"GMS", 84},
		{"GMS", 87},
		{"GMS", 92},
		{"GMS", 95},
		{"JMS", 185},
	}
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	for _, v := range versionsWithJobId {
		got := test.Encode(t, test.CreateContext(v.region, v.major, 1), in.Encode, nil)
		if !bytes.Equal(got, dragonSpawnBody) {
			t.Errorf("%s v%d: bytes = % X, want % X", v.region, v.major, got, dragonSpawnBody)
		}
	}

	got := test.Encode(t, test.CreateContext("GMS", 83, 1), in.Encode, nil)
	if !bytes.Equal(got, dragonSpawnBodyV83) {
		t.Errorf("GMS v83: bytes = % X, want % X", got, dragonSpawnBodyV83)
	}
}

// RoundTrip fails on unconsumed trailing bytes, so this also proves the decoder
// reads the discarded short.
func TestDragonSpawnRoundTrip(t *testing.T) {
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	var out DragonSpawn
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1), in.Encode, out.Decode, nil)
}

// RoundTrip on v83 proves the decoder consumes the entire 11-byte body (through
// the discarded short) with nothing left over — no trailing jobId read.
func TestDragonSpawnRoundTripV83(t *testing.T) {
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	var out DragonSpawn
	test.RoundTrip(t, test.CreateContext("GMS", 83, 1), in.Encode, out.Decode, nil)
	if out.OwnerCharacterId() != 4242 || out.X() != 100 || out.Y() != -200 || out.Stance() != 3 {
		t.Fatalf("v83 round-trip mismatch: %+v", out)
	}
	if out.JobId() != 0 {
		t.Fatalf("v83 round-trip should not populate jobId (absent on the wire), got %d", out.JobId())
	}
}

func TestDragonSpawnDecodeRecoversEveryField(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 1)
	var out DragonSpawn
	test.RoundTrip(t, ctx, NewDragonSpawn(4242, 100, -200, 3, 2214).Encode, out.Decode, nil)
	// RoundTrip decodes into out via the pointer receiver.
	if out.OwnerCharacterId() != 4242 || out.X() != 100 || out.Y() != -200 ||
		out.Stance() != 3 || out.JobId() != 2214 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
