package model

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// sampleAttackInfo builds a representative client->server attack request. skillId
// is 0 (a plain weapon swing) so the keydown/charging and special-skill
// (NightWalker/ThunderBreaker) branches stay quiet and the structure is driven
// purely by attackType + tenant version.
func sampleAttackInfo(at AttackType) *AttackInfo {
	ai := NewAttackInfo(at)
	ai.SetHits(2)
	ai.SetDamage(1)
	ai.SetSkillId(0)
	ai.SetOption(0x10)
	ai.SetLeft(true)
	ai.SetAttackAction(0x05)
	ai.SetActionSpeed(4)
	di := NewDamageInfo(2)
	di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{1000, 2000})
	ai.AddDamageInfo(*di)
	if at == AttackTypeRanged {
		ai.SetBulletPosition(100, 200)
	}
	return ai
}

// TestAttackInfoRoundTrip pins Encode/Decode symmetry for every attack type
// across all tenant variants. RoundTrip fails if any byte is left unconsumed,
// which is exactly what a version-gate drift between Encode and Decode produces
// (e.g. the primary dr-block must be present on BOTH sides for GMS v84+).
func TestAttackInfoRoundTrip(t *testing.T) {
	types := []struct {
		name string
		at   AttackType
	}{
		{"Melee", AttackTypeMelee},
		{"Ranged", AttackTypeRanged},
		{"Magic", AttackTypeMagic},
		{"Energy", AttackTypeEnergy},
	}
	for _, v := range pt.Variants {
		for _, ty := range types {
			t.Run(v.Name+"/"+ty.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				ai := sampleAttackInfo(ty.at)
				pt.RoundTrip(t, ctx, ai.Encode, ai.Decode, nil)
			})
		}
	}
}

// TestAttackInfoSpiritJavelinStarId pins the Shadow Stars / Spirit Javelin star
// id that rides a ranged attack when mask1 bit 6 is set. The gate is mask1 bit 6
// in EVERY client version (verified in the GMS clients v48–v95; jms v185 follows
// v87) — NOT the GMS v95 explicit ExJablin bool. Gating on the wrong flag leaves
// the per-mob damage-info loop reading 4 bytes off, silently dropping all monster
// damage while Shadow Stars is active. RoundTrip byte-balance alone does NOT catch
// that (Encode and Decode drop the field symmetrically), so this asserts the
// decoded star id AND the trailer that sits after it (bulletX/bulletY).
func TestAttackInfoSpiritJavelinStarId(t *testing.T) {
	const starId = uint32(2070006) // an ilbi throwing star (207xxxx)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			ai := sampleAttackInfo(AttackTypeRanged)
			ai.SetOption(0x10 | 0x40) // keep the sample's flags; add Spirit Javelin (bit 6)
			ai.SetBulletItemId(starId)

			l, _ := testlog.NewNullLogger()
			wire := ai.Encode(l, ctx)(nil)

			req := request.Request(wire)
			reader := request.NewRequestReader(&req, 0)
			got := NewAttackInfo(AttackTypeRanged)
			got.Decode(l, ctx)(&reader, nil)

			if !got.SpiritJavelin() {
				t.Fatalf("SpiritJavelin() = false, want true (mask1 bit 6 set)")
			}
			if got.BulletItemId() != starId {
				t.Fatalf("BulletItemId() = %d, want %d — star id dropped or misaligned", got.BulletItemId(), starId)
			}
			// The per-mob damage-info loop sits immediately after the star id — the
			// exact bytes bug #3 garbled. If the star id was not consumed, the loop
			// decodes 4 bytes early and monsterId reads from the star id's bytes.
			di := got.DamageInfo()
			if len(di) != 1 || di[0].MonsterId() != 9001 {
				t.Fatalf("DamageInfo monsterId = %v, want [9001] — damage loop misaligned past the star id", di)
			}
			if reader.Available() > 0 {
				t.Fatalf("%d unconsumed bytes after decode", reader.Available())
			}
		})
	}
}

// TestAttackInfoVersionBoundary pins the corrected dr-block boundary. The
// primary damage-randomizer block (dr0/dr1/dr2/dr3/randomDr/crc32 = 6x uint32)
// is present GMS v84+, NOT v95+ (the bug this fixes was a >=95 gate that left
// v84 attacks reading skillId from the wrong offset -> 0xFFFFFFFF). v84..v94 are
// identical; v95 adds skillLevel(1) + anotherCrc(4) + a per-type int(4) = +9.
func TestAttackInfoVersionBoundary(t *testing.T) {
	enc := func(major uint16, at AttackType) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		ai := sampleAttackInfo(at)
		return pt.Encode(t, ctx, ai.Encode, nil)
	}

	v83 := enc(83, AttackTypeMelee)
	v84 := enc(84, AttackTypeMelee)
	if len(v84) != len(v83)+24 {
		t.Errorf("v84 melee (%d) must be v83 (%d) + 24 bytes (primary dr-block: 6x uint32)", len(v84), len(v83))
	}
	for _, major := range []uint16{85, 86, 87, 94} {
		if got := enc(major, AttackTypeMelee); len(got) != len(v84) {
			t.Errorf("v%d melee (%d) must equal v84 (%d): no structure change until v95", major, len(got), len(v84))
		}
	}
	if v95 := enc(95, AttackTypeMelee); len(v95) != len(v84)+9 {
		t.Errorf("v95 melee (%d) must be v84 (%d) + 9 bytes (skillLevel + anotherCrc + per-type int)", len(v95), len(v84))
	}
}

// TestAttackInfoMagicVersionBoundary pins the MAGIC attack's own version
// boundaries, which TestAttackInfoVersionBoundary above does not exercise --
// it only encodes AttackTypeMelee, which is precisely why two magic-only gate
// errors survived in the codec.
//
// Both were found by reading each client's magic sender (CUserLocal::
// TryDoingMagicAttack) from its COutPacket ctor; the per-version Encode
// addresses are recorded on gmsMagicSecondaryDrBlock / gmsMagicTrailingWord.
//
//   - The secondary dr-block (6x uint32 = 24 bytes) was gated >= 95 but is
//     present from v84. v84/v87/v92 therefore decoded 24 bytes short from
//     skillDataCrc onward, so the per-mob loop read garbage monster ids and
//     every magic attack dealt zero damage.
//   - The magic trailing word (1x uint32) was gated >= 95 but is present from
//     v92, leaving v92 a further 4 bytes short.
//
// Deltas rather than one absolute number per version, so the test states the
// structural claim ("v87 magic == v84 magic") instead of restating the codec.
func TestAttackInfoMagicVersionBoundary(t *testing.T) {
	enc := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		ai := sampleAttackInfo(AttackTypeMagic)
		return pt.Encode(t, ctx, ai.Encode, nil)
	}

	v83, v84 := enc(83), enc(84)
	// 24 primary dr-block + 24 secondary dr-block.
	if len(v84) != len(v83)+48 {
		t.Errorf("v84 magic (%d) must be v83 (%d) + 48 bytes (primary dr-block 24 + secondary dr-block 24)", len(v84), len(v83))
	}
	// v84..v91 magic are structurally identical; v87 is the version the field
	// report came from, so it is named explicitly rather than folded into a loop.
	if v87 := enc(87); len(v87) != len(v84) {
		t.Errorf("v87 magic (%d) must equal v84 (%d): no magic structure change between them", len(v87), len(v84))
	}
	for _, major := range []uint16{85, 86, 91} {
		if got := enc(major); len(got) != len(v84) {
			t.Errorf("v%d magic (%d) must equal v84 (%d)", major, len(got), len(v84))
		}
	}
	// v92 adds the trailing word only.
	v92 := enc(92)
	if len(v92) != len(v84)+4 {
		t.Errorf("v92 magic (%d) must be v84 (%d) + 4 bytes (magic trailing word)", len(v92), len(v84))
	}
	// v95 adds skillLevel(1) + anotherCrc(4) on top of v92; the trailing word
	// is already counted there, so the step is 5, not 9 as it is for melee.
	if v95 := enc(95); len(v95) != len(v92)+5 {
		t.Errorf("v95 magic (%d) must be v92 (%d) + 5 bytes (skillLevel + anotherCrc)", len(v95), len(v92))
	}
}

// TestAttackInfoMagicV87DecodeAlignment is the assertion that actually fails
// for the bug the field report hit. The round-trip tests cannot catch it:
// Encode and Decode shared the same wrong gate, so they agreed with each other
// and disagreed only with the client.
//
// It hand-builds a v87 magic attack in the client's field order (per the IDA
// read recorded on gmsMagicSecondaryDrBlock) and asserts the decoder lands on
// the right monster id. With the secondary dr-block unread, the per-mob loop
// starts 24 bytes early and monsterId decodes out of the dr words -- exactly
// the out-of-range ids atlas-monsters logged as "Unable to get monster [N]".
func TestAttackInfoMagicV87DecodeAlignment(t *testing.T) {
	const (
		wantMonsterId = uint32(1000021) // a real live mob id from the field report
		wantDamage    = uint32(4242)
	)
	ctx := pt.CreateContext("GMS", 87, 1)

	l, _ := testlog.NewNullLogger()
	// The project's own writer, so this pins FIELD ORDER and WIDTHS -- what the
	// IDA read establishes -- without also re-deriving byte order.
	w := response.NewWriter(l)
	w.WriteByte(0x00)      // fieldKey
	w.WriteInt(0x11111111) // dr0
	w.WriteInt(0x22222222) // dr1
	w.WriteByte(0x11)      // hits=1 (low nibble), mob count=1 (high nibble)
	w.WriteInt(0x33333333) // dr2
	w.WriteInt(0x44444444) // dr3
	w.WriteInt(0)          // skillId 0 -> no keydown field
	w.WriteInt(0x55555555) // randomDr
	w.WriteInt(0x66666666) // crc32
	w.WriteInt(0x77777777) // 2dr0   <- the block the >=95 gate skipped
	w.WriteInt(0x88888888) // 2dr1
	w.WriteInt(0x99999999) // 2dr2
	w.WriteInt(0xAAAAAAAA) // 2dr3
	w.WriteInt(0xBBBBBBBB) // 2rnd
	w.WriteInt(0xCCCCCCCC) // 2crc
	w.WriteInt(0xDDDDDDDD) // skillDataCrc
	w.WriteInt(0xEEEEEEEE) // skillDataCrc2
	w.WriteByte(0x00)      // mask1
	w.WriteShort(0x0005)   // mask2
	w.WriteByte(0x01)      // attackActionType
	w.WriteByte(0x04)      // attackSpeed
	w.WriteInt(0x0000BEEF) // attackTime
	// no magic trailing word at v87
	w.WriteInt(wantMonsterId) // --- DamageInfo ---
	w.WriteByte(0x07)         // hitAction
	w.WriteByte(0x00)         // forceAction
	w.WriteByte(0x00)         // frameIdx
	w.WriteByte(0x00)         // calcDamageStatIndex
	w.WriteShort(0)           // hitPositionX
	w.WriteShort(0)           // hitPositionY
	w.WriteShort(0)           // previousPositionX
	w.WriteShort(0)           // previousPositionY
	w.WriteShort(0)           // delay
	w.WriteInt(wantDamage)    // one damage line (hits nibble = 1)
	w.WriteInt(0)             // per-mob crc (GMS v61+)
	w.WriteShort(100)         // characterX
	w.WriteShort(200)         // characterY
	w.WriteByte(0x00)         // dragon = false

	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	got := NewAttackInfo(AttackTypeMagic)
	got.Decode(l, ctx)(&reader, nil)

	dis := got.DamageInfo()
	if len(dis) != 1 {
		t.Fatalf("decoded %d damage entries, want 1 — per-mob loop misaligned", len(dis))
	}
	if dis[0].MonsterId() != wantMonsterId {
		t.Fatalf("monsterId = %d, want %d — decode is misaligned against the v87 client (this is the no-damage bug)", dis[0].MonsterId(), wantMonsterId)
	}
	if d := dis[0].Damages(); len(d) != 1 || d[0] != wantDamage {
		t.Fatalf("damages = %v, want [%d]", d, wantDamage)
	}
	if reader.Available() > 0 {
		t.Fatalf("%d unconsumed bytes after decode — v87 magic layout is wrong", reader.Available())
	}
}

// mesoExplosionAttackInfo builds a meso-explosion (4211006) melee attack.
// Fixture values are chosen to catch nibble-misuse (design §2.2): hits is 0 —
// the client writes nMaxAttackCount & 0xF there, which wraps to 0 at
// attackCount 16 — while the per-mob damage-line counts are 3 and 1. A decoder
// that sizes damage arrays from the nibble reads zero damages and fails the
// round-trip with unconsumed bytes.
func mesoExplosionAttackInfo() *AttackInfo {
	ai := NewAttackInfo(AttackTypeMelee)
	ai.SetDamage(2) // mob count (high nibble)
	ai.SetHits(0)   // nMaxAttackCount & 0xF for attackCount=16 → wrapped to 0
	ai.SetSkillId(4211006)
	ai.SetOption(0)
	ai.SetLeft(false)
	ai.SetAttackAction(0x05)
	ai.SetActionSpeed(4)
	di := NewMesoExplosionDamageInfo()
	di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{100, 200, 300})
	ai.AddDamageInfo(*di)
	di2 := NewMesoExplosionDamageInfo()
	di2.SetMonsterId(9002).SetHitAction(0x06).SetDamages([]uint32{400})
	ai.AddDamageInfo(*di2)
	ai.SetExplodedMesoDrops([]ExplodedMesoDrop{
		NewExplodedMesoDrop(501001, 0x01),
		NewExplodedMesoDrop(501002, 0x03),
	})
	ai.SetMesoDelay(120)
	return ai
}

// TestAttackInfoMesoExplosionRoundTrip pins the meso-explosion variant
// (task-150 design §2.1): per-mob count byte instead of the int16 delay,
// trailing exploded-drop list, trailing int16 delay. The deltas are
// version-invariant; surrounding fields keep their existing gates, which is
// what running across all pt.Variants proves.
func TestAttackInfoMesoExplosionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			ai := mesoExplosionAttackInfo()

			out := NewAttackInfo(AttackTypeMelee)
			pt.RoundTrip(t, ctx, ai.Encode, out.Decode, nil)

			ids := out.ExplodedMesoDrops()
			if len(ids) != 2 || ids[0] != 501001 || ids[1] != 501002 {
				t.Errorf("exploded meso drop ids = %v, want [501001 501002]", ids)
			}
			if out.MesoDelay() != 120 {
				t.Errorf("meso delay = %d, want 120", out.MesoDelay())
			}
			dis := out.DamageInfo()
			if len(dis) != 2 || len(dis[0].Damages()) != 3 || len(dis[1].Damages()) != 1 {
				t.Fatalf("per-mob damage counts wrong: got %d entries", len(dis))
			}
			entries := out.ExplodedMesoDropEntries()
			if len(entries) != 2 || entries[0].HitMask() != 0x01 || entries[1].HitMask() != 0x03 {
				t.Errorf("hit masks did not round-trip: %+v", entries)
			}
			enc1 := pt.Encode(t, ctx, ai.Encode, nil)
			enc2 := pt.Encode(t, ctx, out.Encode, nil)
			if !bytes.Equal(enc1, enc2) {
				t.Errorf("re-encode mismatch:\n got % x\nwant % x", enc2, enc1)
			}
		})
	}
}

// TestAttackInfoNonMesoHasNoDropList pins FR-3's "empty for non-meso attacks"
// contract and that the variant tail is never written for other skills.
func TestAttackInfoNonMesoHasNoDropList(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	ai := sampleAttackInfo(AttackTypeMelee)
	out := NewAttackInfo(AttackTypeMelee)
	pt.RoundTrip(t, ctx, ai.Encode, out.Decode, nil)
	if len(out.ExplodedMesoDrops()) != 0 {
		t.Errorf("non-meso attack decoded %d exploded drops, want 0", len(out.ExplodedMesoDrops()))
	}
}

// TestAttackInfoKeydownField pins which skills cause the attack codec to emit the
// extra tKeyDown u32 (attack_info.go:113 encode / :232 decode). A keydown skill's
// encoded attack is exactly 4 bytes longer than the same attack with a non-keydown
// skillId — the field the v83 client writes at 0x955223 (design §2.3). This guards
// skill.IsKeyDownSkill against accidental broadening (Explosion/Chakra) or narrowing
// (dropping Corkscrew/Grenade), and runs across every tenant variant (all 11 in
// pt.Variants: v28/v48/v61/v72/v79/v83/v84/v86/v87/v95/jms185) so the cross-version
// safety of design §2.4 is enforced in CI. The baseline (skillId 0) and the two
// DROPPED skills are charge:false and non-keydown, so they carry NO keyDown field;
// Hurricane/Corkscrew/Grenade are keydown and add exactly 4 bytes. NOTE: this asserts
// Atlas's own version-agnostic self-consistency (skillId N adds 4 bytes iff
// IsKeyDownSkill(N)), not per-version client faithfulness — for the pre-pirate v28/v48
// contexts the survivors are never actually cast (design §2.4), so the +4 there is a
// harmless unreachable path, and the assertion's value is guarding the reachable versions.
func TestAttackInfoKeydownField(t *testing.T) {
	encLen := func(t *testing.T, region string, major uint16, skillId uint32) int {
		ctx := pt.CreateContext(region, major, 1)
		ai := sampleAttackInfo(AttackTypeMelee)
		ai.SetSkillId(skillId)
		ai.SetKeydown(0xAABBCCDD)
		return len(pt.Encode(t, ctx, ai.Encode, nil))
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			base := encLen(t, v.Region, v.MajorVersion, 0) // skillId 0: not keydown, not charging

			// Non-keydown skills must NOT add a keyDown field (== base length).
			for _, id := range []uint32{
				uint32(skill.FirePoisonMagicianExplosionId), // 2111002 — DROPPED, not keydown
				uint32(skill.ChiefBanditChakraId),           // 4211001 — DROPPED, not keydown
			} {
				if got := encLen(t, v.Region, v.MajorVersion, id); got != base {
					t.Errorf("skill %d: encoded len %d, want %d (non-keydown must carry no tKeyDown field)", id, got, base)
				}
			}

			// Keydown skills add exactly 4 bytes (the tKeyDown u32).
			for _, id := range []uint32{
				uint32(skill.BowmasterHurricaneId),   // 3121004 — pre-existing keydown (guards field didn't move)
				uint32(skill.BrawlerCorkscrewBlowId), // 5101004 — NEW keydown (task-161)
				uint32(skill.GunslingerGrenadeId),    // 5201002 — NEW keydown (task-161)
			} {
				if got := encLen(t, v.Region, v.MajorVersion, id); got != base+4 {
					t.Errorf("skill %d: encoded len %d, want %d (keydown must add a tKeyDown u32 = base+4)", id, got, base+4)
				}
			}
		})
	}
}
