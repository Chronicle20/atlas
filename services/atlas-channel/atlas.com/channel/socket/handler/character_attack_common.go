package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/skill"
	skill2 "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"

	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
)

// computeReflect computes the damage that should be reflected back to the
// attacker for one attack damage entry. dx/dy bounds-check (attacker minus
// monster) is inclusive on every edge so a hit on the LtX/LtY/RbX/RbY edge
// still triggers reflect — matching classic v83 behaviour. The reflected
// total is the sum of all damage lines multiplied by Percent and clamped
// to MaxDamage. Kind matching is the caller's responsibility (see
// monster.StatusMirror.GetReflect).
func computeReflect(damages []int32, info monster.ReflectInfo, attackerX, attackerY, monsterX, monsterY int16) (reflected int32, withinRange bool) {
	dx := attackerX - monsterX
	dy := attackerY - monsterY
	if dx < info.LtX || dx > info.RbX || dy < info.LtY || dy > info.RbY {
		return 0, false
	}
	total := int32(0)
	for _, d := range damages {
		total += d
	}
	r := total * info.Percent / 100
	if r > info.MaxDamage {
		r = info.MaxDamage
	}
	return r, true
}

// attackKindFromAttackType maps a packet AttackType to the reflect kind the
// monster's reflect would have to match for the attack to be reflected.
// Returns the empty string for attack types that cannot be reflected
// (e.g. ENERGY).
func attackKindFromAttackType(at packetmodel.AttackType) string {
	switch at {
	case packetmodel.AttackTypeMelee, packetmodel.AttackTypeRanged:
		return monster2.ReflectKindPhysical
	case packetmodel.AttackTypeMagic:
		return monster2.ReflectKindMagical
	}
	return ""
}

func processAttack(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(ai packetmodel.AttackInfo) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(ai packetmodel.AttackInfo) model.Operator[session.Model] {
		return func(wp writer.Producer) func(ai packetmodel.AttackInfo) model.Operator[session.Model] {
			return func(ai packetmodel.AttackInfo) model.Operator[session.Model] {
				return func(s session.Model) error {
					cp := character.NewProcessor(l, ctx)
					c, err := cp.GetById(cp.InventoryDecorator, cp.SkillModelDecorator)(s.CharacterId())
					if err != nil {
						return err
					}

					var sk skill.Model
					var se effect.Model

					if ai.SkillId() > 0 {
						// Process skill
						for _, tsk := range c.Skills() {
							if tsk.Id() == skill3.Id(ai.SkillId()) {
								sk = tsk
							}
						}
						if sk.Id() == 0 {
							l.Errorf("Character [%d] attempting to attack with skill [%d] which they do not own.", s.CharacterId(), ai.SkillId())
							return session.NewProcessor(l, ctx).Destroy(s)
						}

						se, err = skill2.NewProcessor(l, ctx).GetEffect(ai.SkillId(), sk.Level())
						if err != nil {
							return err
						}
						if se.HPConsume() > 0 {
							_ = cp.ChangeHP(s.Field(), s.CharacterId(), -int16(se.HPConsume()))
						}
						if se.MPConsume() > 0 {
							_ = cp.ChangeMP(s.Field(), s.CharacterId(), -int16(se.MPConsume()))
						}
					}

					// Compute projectile consumption plan before broadcasting so planner
					// errors surface before visible side effects. Emission happens post-broadcast.
					pp := NewProjectileProcessor(l, ctx)
					projectilePlan, hasProjectilePlan := pp.Plan(c, ai, se)

					mp := monster.NewProcessor(l, ctx)
					mirror := monster.GetStatusMirror()
					t := tenant.MustFromContext(ctx)
					attackKind := attackKindFromAttackType(ai.AttackType())

					for _, di := range ai.DamageInfo() {
						damages := di.Damages()
						if len(damages) == 0 {
							// Still allow status apply path (matches prior behaviour
							// of looping into the apply block on empty damage).
							if len(se.MonsterStatus()) > 0 {
								ms := make(map[string]int32)
								for k, v := range se.MonsterStatus() {
									ms[k] = int32(v)
								}
								_ = mp.ApplyStatus(s.Field(), di.MonsterId(), s.CharacterId(), uint32(ai.SkillId()), uint32(sk.Level()), ms, uint32(se.Duration()))
							}
							continue
						}

						reflected := false
						if attackKind != "" {
							if info, ok := mirror.GetReflect(t, di.MonsterId(), attackKind); ok {
								mon, mErr := mp.GetById(di.MonsterId())
								if mErr == nil {
									entry := make([]int32, 0, len(damages))
									for _, d := range damages {
										entry = append(entry, int32(d))
									}
									r, within := computeReflect(entry, info, c.X(), c.Y(), mon.X(), mon.Y())
									if within {
										l.Debugf("reflect: char [%d] hit monster [%d] for %d reflected damage.", s.CharacterId(), di.MonsterId(), r)
										if eErr := mp.EmitDamageReflected(s.Field(), di.MonsterId(), mon.MonsterId(), s.CharacterId(), uint32(r), info.Kind); eErr != nil {
											l.WithError(eErr).Errorf("Unable to emit DAMAGE_REFLECTED for monster [%d] / character [%d].", di.MonsterId(), s.CharacterId())
										}
										reflected = true
									}
								}
							}
						}

						if !reflected {
							if err := mp.Damage(s.Field(), di.MonsterId(), s.CharacterId(), damages, byte(ai.AttackType())); err != nil {
								l.WithError(err).Errorf("Unable to apply damage to monster [%d] from character [%d].", di.MonsterId(), s.CharacterId())
							}
						}

						// Apply monster status effects from skill (e.g., freeze, poison, stun)
						if len(se.MonsterStatus()) > 0 {
							ms := make(map[string]int32)
							for k, v := range se.MonsterStatus() {
								ms[k] = int32(v)
							}
							_ = mp.ApplyStatus(s.Field(), di.MonsterId(), s.CharacterId(), uint32(ai.SkillId()), uint32(sk.Level()), ms, uint32(se.Duration()))
						}
					}

					_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
						var writerName string
						var bodyProducer packet.Encode
						if ai.AttackType() == packetmodel.AttackTypeMelee {
							writerName = charpkt.CharacterAttackMeleeWriter
							bodyProducer = writer.CharacterAttackMeleeBody(c, ai)
						} else if ai.AttackType() == packetmodel.AttackTypeRanged {
							writerName = charpkt.CharacterAttackRangedWriter
							bodyProducer = writer.CharacterAttackRangedBody(c, ai)
						} else if ai.AttackType() == packetmodel.AttackTypeMagic {
							writerName = charpkt.CharacterAttackMagicWriter
							bodyProducer = writer.CharacterAttackMagicBody(c, ai)
						} else if ai.AttackType() == packetmodel.AttackTypeEnergy {
							writerName = charpkt.CharacterAttackEnergyWriter
							bodyProducer = writer.CharacterAttackEnergyBody(c, ai)
						} else {
							return errors.New("unhandled attack type")
						}

						err = session.Announce(l)(ctx)(wp)(writerName)(bodyProducer)(os)
						if err != nil {
							l.WithError(err).Errorf("Unable to announce character [%d] is attacking to character [%d].", s.CharacterId(), os.CharacterId())
							return err
						}
						return nil
					})

					// Projectile reservation + consume emits run fire-and-forget after the
					// broadcast. Classic semantics: the projectile is expended the moment the
					// server accepts the attack, regardless of broadcast success.
					if hasProjectilePlan {
						if perr := pp.Emit(s.CharacterId(), projectilePlan); perr != nil {
							l.WithError(perr).Errorf("Failed to emit projectile consumption for character [%d].", s.CharacterId())
						}
					}

					// TODO apply cooldown
					// TODO cancel dark sight / wind walk
					// TODO apply combo orbs (add or consume)
					// TODO decrease HP from DragonKnight Sacrifice
					// TODO apply attack effect (heal, mp consumption, dispel, cure all, combo reset, etc)
					// TODO destroy Chief Bandit exploded mesos
					// TODO apply Pick Pocket
					// TODO increase HP from Energy Drain, Vampire, or Drain
					// TODO apply Bandit Steal
					// TODO Fire Demon ice weaken
					// TODO Ice Demon fire weaken
					// TODO Homing Beacon / Bullseye
					// TODO Flame Thrower
					// TODO Snow Charge
					// TODO Hamstring
					// TODO Slow
					// TODO Blind
					// TODO Paladin / White Knight charges
					// TODO Combo Drain
					// TODO Mortal Blow
					// TODO Three Snails consumption
					// TODO Heavens Hammer
					// TODO ComboTempest
					// TODO BodyPressure
					// TODO Apply MPEater

					return nil
				}
			}
		}
	}
}
