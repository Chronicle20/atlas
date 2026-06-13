package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSummonSpawn(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0 /*MovementStationary*/, true /*puppet*/, false /*animated*/)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}

// v83 body shared by both the v83 and v95 byte assertions. Per
// summon-packet-delta.md §3.1:
//   int ownerId=42, int oid=0x000F4241, int skillId=3111002=0x002F785A,
//   byte 0x0A (charLevel, visual-only), byte level=20, short x=100, short y=-50,
//   byte stance=0, short 0 (foothold, visual-only), byte movementType=0,
//   bool !puppet=!true=0, bool !animated=!false=1.
var summonSpawnV83Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // ownerId
	0x41, 0x42, 0x0F, 0x00, // oid
	0x5A, 0x78, 0x2F, 0x00, // skillId
	0x0A,       // charLevel (visual-only)
	0x14,       // level=20
	0x64, 0x00, // x=100
	0xCE, 0xFF, // y=-50
	0x00,       // stance
	0x00, 0x00, // foothold (visual-only)
	0x00, // movementType
	0x00, // !puppet
	0x01, // !animated
}

// TestSummonSpawnBytesV83 pins the classic (pre-95) layout. v83/v84/v87/JMS185
// all share this exact byte sequence (summon-packet-delta.md §3.1).
func TestSummonSpawnBytesV83(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonSpawnV83Body)
	}
}

// TestSummonSpawnBytesV95 pins the v95+ DELTA (gated >= 95, GMS only): the v83
// body followed by a single bAvatarLook-present byte = 0. For our 21-summon
// v83 roster no avatar look is carried and Tesla Coil is out of roster, so no
// AvatarLook blob / triangle tail follows (summon-packet-delta.md §3.1,
// CSummoned::Init@0x755740).
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v95 ida=0x75a9a0
func TestSummonSpawnBytesV95(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	want := append(append([]byte{}, summonSpawnV83Body...), 0x00) // + bAvatarLook present = 0
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
	if len(got) != len(summonSpawnV83Body)+1 {
		t.Fatalf("v95 len = %d, want v83 len + 1 = %d", len(got), len(summonSpawnV83Body)+1)
	}
}

// TestSummonSpawnJMS185NoDelta confirms JMS185 (major 185) stays on the classic
// body despite MajorAtLeast(95) being true — the >=95 delta is GMS-gated
// because JMS185's classic reader does NOT decode an avatar-look blob
// (summon-packet-delta.md §3.1 "≥87 gate").
func TestSummonSpawnJMS185NoDelta(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("JMS", 185, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV83Body) {
		t.Fatalf("JMS185 bytes = % X, want classic body % X", got, summonSpawnV83Body)
	}
}
