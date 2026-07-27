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
