package writer

import (
	"atlas-channel/character"
	"atlas-channel/character/skill"
	skill3 "atlas-channel/data/skill"
	"atlas-channel/socket/model"
	"context"
	"math"

	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-constants/job"
	skill2 "github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func WriteCommonAttackBody(l logrus.FieldLogger) func(ctx context.Context) func(c character.Model, ai model.AttackInfo) func(w *response.Writer) {
	return func(ctx context.Context) func(c character.Model, ai model.AttackInfo) func(w *response.Writer) {
		t := tenant.MustFromContext(ctx)
		return func(c character.Model, ai model.AttackInfo) func(w *response.Writer) {
			return func(w *response.Writer) {
				w.WriteInt(c.Id())
				w.WriteByte(byte(ai.Damage()<<4 | uint32(ai.Hits())))
				w.WriteByte(c.Level())
				if ai.SkillId() > 0 {
					// Cannot rely on ai.SkillLevel here because it is not provided GMS < 95
					var sk skill.Model
					for _, tsk := range c.Skills() {
						if tsk.Id() == skill2.Id(ai.SkillId()) {
							sk = tsk
						}
					}
					w.WriteByte(sk.Level())
					w.WriteInt(ai.SkillId())
				} else {
					w.WriteByte(0)
				}
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					// TODO identify what was introduced here
					if skill2.Is(skill2.Id(ai.SkillId()), skill2.SniperStrafeId) {
						w.WriteByte(0) // passive SLV
						if false {
							w.WriteInt(0) // passive skill id
						}
					}
				}

				// TODO better document what this value
				// interested in & 0x20 here
				w.WriteByte(ai.Option())
				left := 0
				if ai.Left() {
					left = 1
				}
				w.WriteInt16(int16((left << 15) | ai.AttackAction()))
				// TODO identify 0x110 constant introduced between GMS v83 and v95
				if ai.AttackAction() <= 0x110 {
					w.WriteByte(ai.ActionSpeed())

					nMastery := byte(0)
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
						nMastery = computeMasteryForWeapon(l)(ctx)(weaponId, c.JobId(), skill2.Id(ai.SkillId()), c.Skills())
					}

					w.WriteByte(nMastery) // mastery

					var bulletItemId = ai.BulletItemId()
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
					w.WriteInt(bulletItemId)

					for _, di := range ai.DamageInfo() {
						w.WriteInt(di.MonsterId())
						if di.MonsterId() > 0 {
							w.WriteByte(di.HitAction())
							if skill2.Is(skill2.Id(ai.SkillId()), skill2.ChiefBanditMesoExplosionId) {
								w.WriteByte(byte(len(di.Damages())))
							}
							for _, d := range di.Damages() {
								w.WriteInt(d)
							}
						}
					}
				}

				if ai.AttackType() == model.AttackTypeRanged {
					w.WriteShort(ai.BulletX())
					w.WriteShort(ai.BulletY())
				}
				if skill2.Is(skill2.Id(ai.SkillId()), skill2.FirePoisonArchMagicianBigBangId, skill2.IceLightningArchMagicianBigBangId, skill2.BishopBigBangId, skill2.EvanStage4IceBreathId, skill2.EvanStage7FireBreathId) {
					w.WriteInt(ai.Keydown())
				}
				// TODO Wild Hunter swallow
			}
		}
	}
}

func computeMasteryForWeapon(l logrus.FieldLogger) func(ctx context.Context) func(weaponId uint32, jobId job.Id, attackSkillId skill2.Id, skills []skill.Model) byte {
	return func(ctx context.Context) func(weaponId uint32, jobId job.Id, attackSkillId skill2.Id, skills []skill.Model) byte {
		t := tenant.MustFromContext(ctx)
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
				if job.IsA(jobId, job.BrawlerId, job.MarauderId, job.BuccaneerId) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.BrawlerKnucklerMasteryId, skills)
				} else if job.IsA(jobId, job.ThunderBreakerStage2Id, job.ThunderBreakerStage3Id, job.ThunderBreakerStage4Id) {
					masteryPercent = getMasteryFromSkill(l)(ctx)(10, skill2.ThunderBreakerStage2KnuckleMasteryId, skills)
				}
			} else if wt == item.WeaponTypeGun {
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
			var start = int8(15)
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
