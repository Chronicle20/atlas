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

// summonDamageV83Body is the v83 wire: cid + oid + body, NO trailing dir byte.
// The cid is read upstream by CUserPool::OnUserCommonPacket@0x972401; CSummonedPool::
// OnPacket@0x938dd7 then does one Decode4 = the oid before the damage leaf
// (OnSkill@0x7a6ebe, the HIGHER swapped opcode), which reads attackIdx(b), dmg(i),
// if attackIdx>-2:{templateId(i), bLeft(b)} and nothing after. (The prior "no oid"
// reading missed the upstream cid — see summon-wire-truth.md.)
//
//	cid=42, oid=1000001=0x000F4241, attackIdx 12, damage=1234=0x000004D2,
//	monsterIdFrom=9300018=0x008DE832, bLeft 0
var summonDamageV83Body = []byte{
	0x2A, 0x00, 0x00, 0x00, // cid
	0x41, 0x42, 0x0F, 0x00, // oid=1000001
	0x0C,                   // attackIdx (12)
	0xD2, 0x04, 0x00, 0x00, // damage
	0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom
	0x00, // bLeft
}

// TestSummonDamageBytes pins the v83 layout: cid + oid + body, no trailing dir
// byte (the dir<0 byte belongs to the SERVERBOUND SetDamaged send, not this
// broadcast). (The prior "no oid" reading missed the upstream CUserPool cid read
// — see summon-wire-truth.md.) NOTE: v84/v87/jms inherit this correction; their
// matrix cells need re-verification against the cid-pre-reading dispatcher.
func TestSummonDamageBytes(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonDamageV83Body)
	}
}

// TestSummonDamageBytesV87 pins that v87 is byte-identical to v83 (cid + oid +
// body, no trailing dir byte). NOTE: v87 inherits the oid correction by the same
// dispatcher logic; this cell needs re-verification against the cid-pre-reading
// dispatcher.
func TestSummonDamageBytesV87(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 87, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v87 bytes = % X, want % X (identical to v83)", got, summonDamageV83Body)
	}
}

// TestSummonDamageBytesV95 pins that v95 is byte-identical to v83 for damage: the
// oid is now in the shared body and there is no v95-specific delta (v95 OnHit@
// 0x74bc80 stops at bLeft — the dir byte is serverbound only).
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v95 ida=0x7598c0
func TestSummonDamageBytesV95(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v95 bytes = % X, want % X (identical to v83)", got, summonDamageV83Body)
	}
}
