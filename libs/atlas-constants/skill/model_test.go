package skill

import "testing"

// TestIsKeyDownSkill pins the exact membership of the keydown predicate. The two
// task-161 additions (Corkscrew Blow, Grenade) are IDA-verified keydown in the v83
// client; the two PRD-dropped skills (Explosion, Chakra) are NOT keydown in any
// version and must never be re-added (adding them broadcasts a phantom aura and
// makes attack_info.go over-read a tKeyDown field the client never sends).
func TestIsKeyDownSkill(t *testing.T) {
	keydown := []Id{
		FirePoisonArchMagicianBigBangId,
		IceLightningArchMagicianBigBangId,
		BishopBigBangId,
		HeroMonsterMagnetId,
		PaladinMonsterMagnetId,
		DarkKnightMonsterMagnetId,
		BowmasterHurricaneId,
		MarksmanPiercingArrowId,
		CorsairRapidFireId,
		NightWalkerStage3PoisonBombId,
		WindArcherStage3HurricaneId,
		ThunderBreakerStage2CorkscrewBlowId,
		EvanStage4IceBreathId,
		EvanStage7FireBreathId,
		BrawlerCorkscrewBlowId, // 5101004 — added task-161 (IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185)
		GunslingerGrenadeId,    // 5201002 — added task-161 (IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185)
	}
	for _, id := range keydown {
		if !IsKeyDownSkill(id) {
			t.Errorf("IsKeyDownSkill(%d) = false, want true", uint32(id))
		}
	}

	notKeydown := []Id{
		FirePoisonMagicianExplosionId, // 2111002 — DROPPED (FR-1.4), not keydown in client
		ChiefBanditChakraId,           // 4211001 — DROPPED (FR-1.4), not keydown in client
		FighterFinalAttackAxeId,       // 1100003 — plain non-keydown control
	}
	for _, id := range notKeydown {
		if IsKeyDownSkill(id) {
			t.Errorf("IsKeyDownSkill(%d) = true, want false", uint32(id))
		}
	}
}

// TestNeedsMasterLevelMatchesClientRule pins the port of the client's
// is_skill_need_master_level. The per-version decompiles are recorded on the
// function; this asserts the behaviour that actually broke.
//
// The task-218 field report: a preset Evan (job 2218, 31 skills) closed the
// client with error 38 while a level-1 Evan logged in fine. The server was
// gating the trailing master-level int on job.IsFourthJob, true for the whole
// 2214-2218 band, where the client selects only growths 9-10 (jobs 2217/2218)
// plus three named skills. Every mismatched skill shifts the rest of
// GW_CharacterData by 4 bytes, which is why an Evan with no skills was fine.
func TestNeedsMasterLevelMatchesClientRule(t *testing.T) {
	for _, c := range []struct {
		name       string
		skillId    Id
		exceptions bool
		want       bool
	}{
		// Evan: only the 9th and 10th growths (jobs 2217, 2218).
		{"Evan 9th growth", Id(22171000), true, true},
		{"Evan 10th growth", Id(22181000), true, true},
		// The band the old per-job gate wrongly included.
		{"Evan 6th growth", Id(22141001), true, false},
		{"Evan 7th growth", Id(22151001), true, false},
		{"Evan 8th growth (Recovery Aura)", Id(22161003), true, false},
		{"Evan 5th growth", Id(22131000), true, false},
		{"Evan 1st growth", Id(22001001), true, false},
		{"Evan beginner job 2001", Id(20010000), true, false},
		// The three named exceptions — GMS only.
		{"Magic Guard exception (GMS)", Id(22111001), true, true},
		{"Magic Guard exception (JMS)", Id(22111001), false, false},
		{"22141002 exception (GMS)", Id(22141002), true, true},
		{"22141002 exception (JMS)", Id(22141002), false, false},
		{"Critical Magic exception (GMS)", Id(22140000), true, true},
		{"Critical Magic exception (JMS)", Id(22140000), false, false},
		// Generic branch: 4th job only, job roots excluded. Identical on every
		// client read (v83/v84/v87/v92/v95/jms185).
		{"Hero 4th job", Id(1120003), true, true},
		{"Bishop 4th job", Id(2321000), true, true},
		{"Night Lord 4th job", Id(4120002), true, true},
		{"Fighter 2nd job", Id(1100000), true, false},
		{"Crusader 3rd job", Id(1110000), true, false},
		{"Beginner root job 0", Id(10000), true, false},
		{"Warrior root job 100", Id(1000000), true, false},
		{"Magician root job 200", Id(2000000), true, false},
		// Aran 4th job (2112) goes through the generic branch, not the Evan one.
		{"Aran 4th job", Id(21120001), true, true},
		{"Aran 3rd job", Id(21110000), true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := NeedsMasterLevel(c.skillId, c.exceptions); got != c.want {
				t.Errorf("NeedsMasterLevel(%d, exceptions=%v) = %v, want %v", c.skillId, c.exceptions, got, c.want)
			}
		})
	}
}

// TestNeedsMasterLevelNotSkillBookIndexed guards the specific off-by-one that
// job.GetSkillBook would introduce: its Evan indexing is 2210->1 … 2218->9,
// while the client's jobLevel is 2210->2 … 2218->10. Selecting "book 9 or 10"
// with GetSkillBook would pick jobs 2218 and 2219 instead of 2217 and 2218.
func TestNeedsMasterLevelNotSkillBookIndexed(t *testing.T) {
	if !NeedsMasterLevel(Id(22171000), true) {
		t.Error("job 2217 must need a master level (client jobLevel 9)")
	}
	if NeedsMasterLevel(Id(22161000), true) {
		t.Error("job 2216 must NOT need a master level — picking it means the skill-book index was used")
	}
}
