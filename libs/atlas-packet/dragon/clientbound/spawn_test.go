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

func TestDragonSpawnBytes(t *testing.T) {
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, dragonSpawnBody) {
		t.Fatalf("spawn bytes = % X, want % X", got, dragonSpawnBody)
	}
}

// The layout is uniform across all six applicable versions (verified in both
// client size classes: 0x330 = v83/v87/JMS185, 0x464 = v92/v95). If any column
// ever diverges, this table is where it shows up first.
func TestDragonSpawnBytesIdenticalAcrossVersions(t *testing.T) {
	versions := []struct {
		region string
		major  uint16
	}{
		{"GMS", 83},
		{"GMS", 84},
		{"GMS", 87},
		{"GMS", 92},
		{"GMS", 95},
		{"JMS", 185},
	}
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	for _, v := range versions {
		got := test.Encode(t, test.CreateContext(v.region, v.major, 1), in.Encode, nil)
		if !bytes.Equal(got, dragonSpawnBody) {
			t.Errorf("%s v%d: bytes = % X, want % X", v.region, v.major, got, dragonSpawnBody)
		}
	}
}

// RoundTrip fails on unconsumed trailing bytes, so this also proves the decoder
// reads the discarded short.
func TestDragonSpawnRoundTrip(t *testing.T) {
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	var out DragonSpawn
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1), in.Encode, out.Decode, nil)
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
