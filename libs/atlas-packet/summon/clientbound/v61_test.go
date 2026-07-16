package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 summon clientbound fixtures. Every summon writer is byte-identical to the
// verified GMS v72 wire (v61 is below the >=83 SLV gate and the >=95 avatar-look
// gate, so all summon codecs take the same legacy branch as v72). Live-verified
// against GMS_v61.1_U_DEVM.exe @port 13338.
//
// Dispatch: CUserPool::OnUserCommonPacket reads cid upstream, then the summon
// cluster dispatcher CSummonedPool::OnPacket sub_7922E8@0x7922e8 routes ops
// 134-139. For op 134 (SPAWN) it vtable-calls the create leaf sub_792472@0x792472
// WITHOUT pre-reading (leaf reads oid). For ops 135-139 the else branch first
// reads Decode4(oid)@0x792327 (the pool key), then routes to the per-op leaf.

// TestSummonSpawnBytesV61 pins the v61 spawn wire = v72 (cid, oid, skillId,
// charLevel, Init blob; NO SLV, NO avatar byte). Create leaf sub_792472@0x792472
// reads Decode4(oid)@0x792494, Decode4(skillId)@0x79249e, Decode1(charLevel)@0x7924ad
// (the ONLY byte before the ctor sub_678E39 — NO SLV), then the Init-blob reader
// sub_679087 reads Decode2(x)@0x6790ad, Decode2(y)@0x6790bb, Decode1(stance)@0x6790db,
// Decode2(foothold)@0x6790fe, Decode1(movementType)@0x67910b, Decode1(!puppet)@0x679111,
// and later Decode1(!animated)@0x679459. NO trailing avatar byte on v61.
// packet-audit:verify packet=summon/clientbound/SummonSpawn version=gms_v61 ida=0x792472
func TestSummonSpawnBytesV61(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, 0, true, false)
	ctx := test.CreateContext("GMS", 61, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonSpawnV79Body) {
		t.Fatalf("v61 bytes = % X, want % X (identical to v72)", got, summonSpawnV79Body)
	}
}

// TestSummonRemoveBytesV61 pins the v61 remove wire = v72: cid (upstream) + oid
// (dispatcher Decode4@0x792327) + one flag byte. The op135 path calls the remove
// handler sub_67BFED which reads Decode1@0x67c002 (leave/animated flag, branched
// 0/2/3/4 exactly as v72's sub_6E8F0F). Nothing else.
// packet-audit:verify packet=summon/clientbound/SummonRemove version=gms_v61 ida=0x7922e8
func TestSummonRemoveBytesV61(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	ctx := test.CreateContext("GMS", 61, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // ownerId (cid, consumed upstream)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x792327 in sub_7922E8)
		0x04, // animated ? 4 : 1 (Decode1@0x67c002 in sub_67BFED)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 bytes = % X, want % X", got, want)
	}
}

// TestSummonMoveBytesV61 pins the v61 move wire = v72: cid (upstream) + oid
// (dispatcher Decode4@0x792327) + the raw CMovePath blob. op136 calls the move
// leaf sub_67C37E@0x67c37e → CMovePath::OnMovePacket (the blob begins with the
// start position; it is rebroadcast byte-faithfully, no separate start write).
// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v61 ida=0x67c37e
func TestSummonMoveBytesV61(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	in := NewSummonMove(42, 1000001, raw)
	ctx := test.CreateContext("GMS", 61, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid (consumed upstream)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x792327 in sub_7922E8)
		0x01, 0x02, 0x03, 0x04, 0x05, // rawMovement (CMovePath blob)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 bytes = % X, want % X", got, want)
	}
}

// TestSummonAttackBytesV61 pins the v61 attack wire = v72 (no leading charLevel
// byte; MajorAtLeast(83)=false). op137 calls the attack leaf sub_67C39F@0x67c39f,
// which reads Decode1(action)@0x67c425 (v6=(b>>7)&1 bLeft, v73=b&0x7F direction) —
// NO leading charLevel byte; Decode1(count)@0x67c44a; per target Decode4(mobOid)@0x67c473,
// and if(oid){ Decode1@0x67c481; Decode4(damage)@0x67c494 }. Nothing after the loop.
// packet-audit:verify packet=summon/clientbound/SummonAttack version=gms_v61 ida=0x67c39f
func TestSummonAttackBytesV61(t *testing.T) {
	targets := []SummonAttackTarget{
		NewSummonAttackTarget(1000001, 1234),
		NewSummonAttackTarget(1000002, 5678),
	}
	in := NewSummonAttack(42, 2000001, 3, targets)
	ctx := test.CreateContext("GMS", 61, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonAttackV79Body) {
		t.Fatalf("v61 bytes = % X, want % X (identical to v72)", got, summonAttackV79Body)
	}
	if len(got) != len(summonAttackV83Body)-1 {
		t.Fatalf("v61 len = %d, want v83 len - 1 (no charLevel) = %d", len(got), len(summonAttackV83Body)-1)
	}
}

// TestSummonSkillBytesV61 pins the v61 wire = v72: cid + oid + single stance byte.
// op138 calls the skill leaf sub_67C8D2@0x67c8d2, which reads exactly ONE byte:
// Decode1@0x67c921 then `and eax, 7Fh` (mask 0x7F) → sub_67BB1D. No summonSkillId
// int on the wire.
// packet-audit:verify packet=summon/clientbound/SummonSkill version=gms_v61 ida=0x67c8d2
func TestSummonSkillBytesV61(t *testing.T) {
	in := NewSummonSkill(42, 1000001, 6)
	ctx := test.CreateContext("GMS", 61, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid (consumed upstream)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x792327 in sub_7922E8)
		0x06, // newStance (Decode1@0x67c921, masked 0x7F)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 bytes = % X, want % X", got, want)
	}
}

// TestSummonDamageBytesV61 pins the v61 wire = v72 (cid + oid + body, no trailing
// dir byte). op139 calls the damage leaf sub_67C936@0x67c936, which reads
// Decode1(attackIdx)@0x67c967 (atlas writes 12), Decode4(damage)@0x67c97c; if
// (attackIdx > -2): Decode4(monsterIdFrom)@0x67c98e (→ GetMobTemplate),
// Decode1(bLeft)@0x67c99c (atlas writes 0). Nothing after.
// packet-audit:verify packet=summon/clientbound/SummonDamage version=gms_v61 ida=0x67c936
func TestSummonDamageBytesV61(t *testing.T) {
	in := NewSummonDamage(42, 1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 61, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonDamageV83Body) {
		t.Fatalf("v61 bytes = % X, want % X (identical to v72)", got, summonDamageV83Body)
	}
}
