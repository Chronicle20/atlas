package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSummonDamage(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}

// summonDamageV83Body is the classic v83 wire: NO oid, NO trailing dir byte.
// CSummonedPool::OnSkill@0x7a6ebe (dispatched on the HIGHER swapped opcode) reads
// attackIdx(b), dmg(i), if attackIdx>-2:{templateId(i), bLeft(b)} and nothing
// after — IDB-confirmed (summon-wire-truth.md).
//
//	cid=42, attackIdx 12, damage=1234=0x000004D2,
//	monsterIdFrom=9300018=0x008DE832, bLeft 0
var summonDamageV83Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // cid
	0x0C,                   // attackIdx (12)
	0xD2, 0x04, 0x00, 0x00, // damage
	0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom
	0x00, // bLeft
}

// TestSummonDamageBytes pins the classic v83 layout (no oid, no trailing byte).
// v83 DAMAGE behavior lives at OnSkill@0x7a6ebe (the HIGHER of the swapped
// skill/damage opcodes); the export key CSummonedPool::OnHit records this addr.
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v83 ida=0x7a6ebe
func TestSummonDamageBytes(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonDamageV83Body)
	}
}

// TestSummonDamageBytesV87 pins that v87 is byte-identical to v83: NO oid (oid is
// v95+) and NO trailing dir byte (v87 OnSkill@0x7f969f reads nothing after bLeft,
// same as v83 and v95 — IDB-confirmed). v87 DAMAGE behavior lives at OnSkill@0x7f969f
// (op 0xC0, the higher of the swapped skill/damage handlers); the export key
// CSummonedPool::OnHit records this addr.
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v87 ida=0x7f969f
func TestSummonDamageBytesV87(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 87, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v87 bytes = % X, want % X (identical to v83)", got, summonDamageV83Body)
	}
}

// TestSummonDamageBytesV95 pins the v95+ layout: oid after cid; still NO trailing
// dir byte (v95 OnHit@0x74bc80 stops at bLeft — the dir byte is serverbound only).
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v95 ida=0x7598c0
func TestSummonDamageBytesV95(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=0x000F4241, attackIdx 12, damage, monsterIdFrom, bLeft
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x41, 0x42, 0x0F, 0x00, // oid (v95+ only)
		0x0C,                   // attackIdx (12)
		0xD2, 0x04, 0x00, 0x00, // damage
		0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom
		0x00, // bLeft
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
}
