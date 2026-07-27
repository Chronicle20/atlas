package model

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
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
