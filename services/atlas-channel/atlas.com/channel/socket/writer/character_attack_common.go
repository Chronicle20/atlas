package writer

import (
	"atlas-channel/character"
	"atlas-channel/character/skill"
	skill3 "atlas-channel/data/skill"
	"context"
	"math"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func preComputeAttackValues(l logrus.FieldLogger, ctx context.Context, c character.Model, ai packetmodel.AttackInfo) (skillLevel byte, mastery byte, bulletItemId uint32) {
	// Skill level lookup
	if ai.SkillId() > 0 {
		for _, sk := range c.Skills() {
			if sk.Id() == skill2.Id(ai.SkillId()) {
				skillLevel = sk.Level()
				break
			}
		}
	}

	// Mastery computation
	weaponId := uint32(0)
	ws, err := slot.GetSlotByType("weapon")
	if err == nil {
		if ew, ok := c.Equipment().Get(ws.Type); ok {
			if we := ew.Equipable; we != nil {
				weaponId = we.TemplateId()
			}
		}
	}
	if weaponId > 0 {
		mastery = computeMasteryForWeapon(l)(ctx)(weaponId, c.JobId(), skill2.Id(ai.SkillId()), c.Skills())
	}

	// Bullet resolution
	bulletItemId = ai.BulletItemId()
	if ai.CashBulletPosition() > 0 {
		for _, i := range c.Inventory().Cash().Assets() {
			if uint16(i.Slot()) == ai.CashBulletPosition() {
				bulletItemId = i.TemplateId()
				break
			}
		}
	} else if ai.ProperBulletPosition() > 0 {
		for _, i := range c.Inventory().Consumable().Assets() {
			if uint16(i.Slot()) == ai.ProperBulletPosition() && (item.IsBullet(item.Id(i.TemplateId())) || item.IsThrowingStar(item.Id(i.TemplateId()))) {
				bulletItemId = i.TemplateId()
			}
		}
	}

	return
}

// version-stable per task-187 audit: the Big Bang keydown skills (Magician
// v0.92→v0.95 reorg aside) and the Evan keydown skills do not remap across
// the provisioned GMS versions at these specific wire ids.
func isKeydownSkill(skillId uint32) bool {
	return skill2.Is(skill2.Id(skillId), skill2.FirePoisonArchMagicianBigBangId, skill2.IceLightningArchMagicianBigBangId, skill2.BishopBigBangId, skill2.EvanStage4IceBreathId, skill2.EvanStage7FireBreathId)
}

// computeMasteryForWeapon routes on jobId/attackSkillId to find the weapon
// mastery skill governing the caster's current weapon. Every branch below is
// version-stable per the task-187 audit (Warrior/Page/Fighter/Magician/
// Bowman/Thief/Spearman/Aran roots do not remap across the provisioned GMS
// range) EXCEPT the Knuckle branch: job.BrawlerId/MarauderId/BuccaneerId
// (wire 510/511/512) collide with GM/SuperGM (wire 500/510) at v0.48 — job
// 510 means SuperGM there, not Brawler (divergent set, task-187 audit). That
// branch alone is routed through job identity resolution below; every other
// branch stays Id-keyed with a one-line citation, per the task-187 Task 10
// scope (do not force-migrate version-stable roots).
func computeMasteryForWeapon(l logrus.FieldLogger) func(ctx context.Context) func(weaponId uint32, jobId job.Id, attackSkillId skill2.Id, skills []skill.Model) byte {
	return func(ctx context.Context) func(weaponId uint32, jobId job.Id, attackSkillId skill2.Id, skills []skill.Model) byte {
		t := tenant.MustFromContext(ctx)
		set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
		return func(weaponId uint32, jobId job.Id, attackSkillId skill2.Id, skills []skill.Model) byte {
			masteryPercent := int8(10)
			wt := item.GetWeaponType(item.Id(weaponId))
			if wt == item.WeaponTypeOneHandedSword {
				if job.IsA(jobId, job.FighterId, job.CrusaderId, job.HeroId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.FighterSwordMasteryId, skills)
				} else if job.IsA(jobId, job.PageId, job.WhiteKnightId, job.PaladinId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.PageSwordMasteryId, skills)
				} else if job.IsA(jobId, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.DawnWarriorStage2SwordMasteryId, skills)
				}
			} else if wt == item.WeaponTypeOneHandedAxe {
				if job.IsA(jobId, job.FighterId, job.CrusaderId, job.HeroId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.FighterAxeMasteryId, skills)
				}
			} else if wt == item.WeaponTypeOneHandedMace {
				if job.IsA(jobId, job.PageId, job.WhiteKnightId, job.PaladinId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.PageBluntWeaponMasteryId, skills)
				}
			} else if wt == item.WeaponTypeDagger {
				if job.IsA(jobId, job.BanditId, job.ChiefBanditId, job.ShadowerId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.BanditDaggerMasteryId, skills)
				}
			} else if wt == item.WeaponTypeWand {
				if job.IsA(jobId, job.MagicianId,
					job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
					job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
					job.ClericId, job.PriestId, job.BishopId,
					job.BlazeWizardStage1Id, job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id,
					job.EvanStage1Id, job.EvanStage2Id, job.EvanStage3Id, job.EvanStage4Id, job.EvanStage5Id, job.EvanStage6Id, job.EvanStage7Id, job.EvanStage8Id, job.EvanStage9Id, job.EvanStage10Id) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, attackSkillId, skills)
					// TODO BlazeWizardSpellMastery?
					if job.IsA(job.EvanStage9Id, job.EvanStage10Id) {
						masteryPercent = getMasteryFromSkill(l)(ctx)(masteryPercent, skill2.EvanStage9MagicMasteryId, skills)
					}
				}
			} else if wt == item.WeaponTypeStaff {
				if job.IsA(jobId, job.MagicianId,
					job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
					job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
					job.ClericId, job.PriestId, job.BishopId,
					job.BlazeWizardStage1Id, job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id,
					job.EvanStage1Id, job.EvanStage2Id, job.EvanStage3Id, job.EvanStage4Id, job.EvanStage5Id, job.EvanStage6Id, job.EvanStage7Id, job.EvanStage8Id, job.EvanStage9Id, job.EvanStage10Id) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, attackSkillId, skills)
					// TODO BlazeWizardSpellMastery?
					if job.IsA(job.EvanStage9Id, job.EvanStage10Id) {
						masteryPercent = getMasteryFromSkill(l)(ctx)(masteryPercent, skill2.EvanStage9MagicMasteryId, skills)
					}
				}
			} else if wt == item.WeaponTypeTwoHandedSword {
				if job.IsA(jobId, job.FighterId, job.CrusaderId, job.HeroId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.FighterSwordMasteryId, skills)
				} else if job.IsA(jobId, job.PageId, job.WhiteKnightId, job.PaladinId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.PageSwordMasteryId, skills)
				} else if job.IsA(jobId, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.DawnWarriorStage2SwordMasteryId, skills)
				}
			} else if wt == item.WeaponTypeTwoHandedAxe {
				if job.IsA(jobId, job.FighterId, job.CrusaderId, job.HeroId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.FighterAxeMasteryId, skills)
				}
			} else if wt == item.WeaponTypeTwoHandedMace {
				if job.IsA(jobId, job.PageId, job.WhiteKnightId, job.PaladinId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.PageBluntWeaponMasteryId, skills)
				}
			} else if wt == item.WeaponTypeSpear {
				if job.IsA(jobId, job.SpearmanId, job.DragonKnightId, job.DarkKnightId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.SpearmanSpearMasteryId, skills)
				}
			} else if wt == item.WeaponTypePolearm {
				if job.IsA(jobId, job.SpearmanId, job.DragonKnightId, job.DarkKnightId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.SpearmanPolearmMasteryId, skills)
				} else if job.IsA(jobId, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.AranStage2PolearmMasteryId, skills)
					if job.IsA(jobId, job.AranStage4Id) {
						masteryPercent = getMasteryFromSkill(l)(ctx)(masteryPercent, skill2.AranStage4HighMasteryId, skills)
					}
				}
			} else if wt == item.WeaponTypeBow {
				if job.IsA(jobId, job.HunterId, job.RangerId, job.BowmasterId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.HunterBowMasteryId, skills)
					if job.IsA(jobId, job.BowmasterId) {
						masteryPercent = getMasteryFromSkill(l)(ctx)(masteryPercent, skill2.BowmasterBowExpertId, skills)
					}
				} else if job.IsA(jobId, job.WindArcherStage2Id, job.WindArcherStage3Id, job.WindArcherStage4Id) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.WindArcherStage2BowMasteryId, skills)
				}
			} else if wt == item.WeaponTypeCrossbow {
				if job.IsA(jobId, job.CrossbowmanId, job.SniperId, job.MarksmanId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.CrossbowmanCrossbowMasteryId, skills)
				}
			} else if wt == item.WeaponTypeClaw {
				if job.IsA(jobId, job.AssassinId, job.HermitId, job.NightLordId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.AssassinClawMasteryId, skills)
				} else if job.IsA(jobId, job.NightWalkerStage2Id, job.NightWalkerStage3Id, job.NightWalkerStage4Id) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.NightWalkerStage2ClawMasteryId, skills)
				}
			} else if wt == item.WeaponTypeKnuckle {
				// DIVERGENT (task-187 audit, found beyond the Task 10 brief's
				// listed sites): job.BrawlerId/MarauderId/BuccaneerId are wire
				// 510/511/512 -- wire 510 collides with SuperGM at v0.48
				// (job 500/510 GM/SuperGM vs Pirate/Brawler is the audit's
				// divergent job set). A raw job.IsA(jobId, job.BrawlerId, ...)
				// compare would misclassify a v0.48 SuperGM (wire job 510)
				// wielding a Knuckle as a Brawler. Resolve jobId to its
				// version-blind Identity first.
				if jid, jok := set.Job.Resolve(jobId); jok && job.IsAIdentity(jid, job.Brawler, job.Marauder, job.Buccaneer) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.BrawlerKnucklerMasteryId, skills)
				} else if job.IsA(jobId, job.ThunderBreakerStage2Id, job.ThunderBreakerStage3Id, job.ThunderBreakerStage4Id) {
					// version-stable per task-187 audit: ThunderBreaker (Cygnus
					// 1xxx branch) does not remap across the provisioned GMS
					// versions.
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.ThunderBreakerStage2KnuckleMasteryId, skills)
				}
			} else if wt == item.WeaponTypeGun {
				// version-stable per task-187 audit: Gunslinger/Outlaw/Corsair
				// (wire 520/521/522) are absent -- not aliased -- at v0.48
				// (GM/SuperGM's job tree stops at 500/510), so this is not in
				// the audit's divergent set.
				if job.IsA(jobId, job.GunslingerId, job.OutlawId, job.CorsairId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.GunslingerGunMasteryId, skills)
				}
			}

			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				// calculation is performed in client.
				return byte(masteryPercent)
			} else {
				v := int8(0)
				if masteryPercent-10 <= 0 {
					v = 1
				}
				return byte(((masteryPercent - 10) & (v - 1)) / 5)
			}
		}
	}
}

func getMasteryFromSkill(l logrus.FieldLogger) func(ctx context.Context) func(startingMastery int8, skillId skill2.Id, skills []skill.Model) int8 {
	return func(ctx context.Context) func(startingMastery int8, skillId skill2.Id, skills []skill.Model) int8 {
		return func(startingMastery int8, skillId skill2.Id, skills []skill.Model) int8 {
			start := int8(15)
			// version-stable per task-187 audit: Evan (22xx), Aran (20xx), and
			// Bowmaster (Bowman 3xx branch) high-mastery ids do not remap
			// across the provisioned GMS versions. skillId here is always fed
			// by a caller-supplied canonical constant, never the raw wire id.
			if skill2.Is(skillId, skill2.EvanStage9MagicMasteryId, skill2.AranStage4HighMasteryId, skill2.BowmasterBowExpertId) {
				start = int8(65)
			}

			var s skill.Model
			for _, rs := range skills {
				if rs.Id() == skillId {
					s = rs
				}
			}
			if s.Id() == 0 {
				return startingMastery
			}
			if s.Level() == 0 {
				return startingMastery
			}
			si, err := skill3.NewProcessor(l, ctx).GetById(uint32(skillId))
			if err != nil {
				return startingMastery
			}
			maxLevel := uint32(len(si.Effects()))
			gs := uint32(2)
			if maxLevel == 30 {
				gs = 3
			}
			if start == 65 {
				gs = 5
			}
			return start + 5*int8(math.Floor(float64(s.Level()-1)/float64(gs)))
		}
	}
}
