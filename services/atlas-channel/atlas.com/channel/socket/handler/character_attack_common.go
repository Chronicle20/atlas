package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/skill"
	"atlas-channel/consumable"
	skill2 "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	channelinv "atlas-channel/inventory"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/skill/handler"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"math"
	"math/rand"

	charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	itemconst "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
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

// snapshotVenomDamagePerTick computes the per-tick damage applied by a
// VENOM stack at apply time. Classic formula: round(coef * Luck *
// MagicAttack), where coef is drawn from [0.1, 0.2). The math is pulled
// out of the handler so it can be pinned by unit tests; the production
// site picks the coef via rand.Float64() and feeds the result here.
func snapshotVenomDamagePerTick(luck, magicAttack int, coef float64) int32 {
	return int32(math.Round(coef * float64(luck) * float64(magicAttack)))
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

// findItemSlotInInventory returns the slot of the first asset in the
// compartment matching the item's inventory type whose template id equals
// itemId. Returns false if the inventory has no such item. Used by
// processAttack to translate an effect.ItemConsume() id into a slot
// position before emitting RequestItemConsume.
func findItemSlotInInventory(inv channelinv.Model, itemId uint32) (slot.Position, bool) {
	invType, ok := inventoryconst.TypeFromItemId(itemconst.Id(itemId))
	if !ok {
		return 0, false
	}
	comp := inv.CompartmentByType(invType)
	for _, a := range comp.Assets() {
		if a.TemplateId() == itemId {
			return slot.Position(a.Slot()), true
		}
	}
	return 0, false
}

// damageInfoEntryDeps groups the per-attack closures and lookups that
// processDamageInfoEntry needs. Wrapping them keeps the helper signature
// readable and lets tests construct fakes with a single struct.
type damageInfoEntryDeps struct {
	getReflect        func(t tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool)
	getMonster        func(monsterId uint32) (monster.Model, error)
	applyDamage       func(f field.Model, monsterId, characterId uint32, damages []uint32, attackType byte) error
	emitReflectDamage func(f field.Model, uniqueId, templateId, characterId uint32, reflectDamage uint32, reflectType string) error
	applyStatus       func(f field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error
	loadVenomStats    func() effective_stats.RestModel
}

// processDamageInfoEntry handles one DamageInfo from a magic/melee/ranged
// attack packet: damage application or reflect emission, then optional
// monster status apply. All side-effecting calls go through deps so tests
// can drive each branch without constructing a real monster.Processor or
// session.
func processDamageInfoEntry(
	l logrus.FieldLogger,
	di packetmodel.DamageInfo,
	ai packetmodel.AttackInfo,
	se effect.Model,
	skillLevel uint32,
	casterId uint32,
	casterX, casterY int16,
	f field.Model,
	t tenant.Model,
	attackKind string,
	deps damageInfoEntryDeps,
) {
	damages := di.Damages()

	if len(damages) == 0 {
		if len(se.MonsterStatus()) == 0 {
			return
		}
		ms := make(map[string]int32)
		for k, v := range se.MonsterStatus() {
			ms[k] = int32(v)
		}
		if _, isVenom := ms["VENOM"]; isVenom {
			stats := deps.loadVenomStats()
			coef := 0.1 + rand.Float64()*0.1
			ms["VENOM"] = snapshotVenomDamagePerTick(int(stats.Luck), int(stats.MagicAttack), coef)
		}

		// Doom: respect magic-reflect. Doom does no damage, so on reflect we
		// simply skip the apply (nothing to bounce back). Gated on DOOM so
		// no other empty-damage status flow changes behavior.
		if _, isDoom := ms["DOOM"]; isDoom && attackKind != "" {
			if _, ok := deps.getReflect(t, di.MonsterId(), attackKind); ok {
				l.Debugf("Doom: monster [%d] has %s reflect; status apply skipped.", di.MonsterId(), attackKind)
				return
			}
		}

		_ = deps.applyStatus(f, di.MonsterId(), casterId, uint32(ai.SkillId()), skillLevel, ms, uint32(se.Duration()))
		return
	}

	reflected := false
	if attackKind != "" {
		if info, ok := deps.getReflect(t, di.MonsterId(), attackKind); ok {
			mon, mErr := deps.getMonster(di.MonsterId())
			if mErr == nil {
				entry := make([]int32, 0, len(damages))
				for _, d := range damages {
					entry = append(entry, int32(d))
				}
				r, within := computeReflect(entry, info, casterX, casterY, mon.X(), mon.Y())
				if within {
					l.Debugf("reflect: char [%d] hit monster [%d] for %d reflected damage.", casterId, di.MonsterId(), r)
					if eErr := deps.emitReflectDamage(f, di.MonsterId(), mon.MonsterId(), casterId, uint32(r), info.Kind); eErr != nil {
						l.WithError(eErr).Errorf("Unable to emit DAMAGE_REFLECTED for monster [%d] / character [%d].", di.MonsterId(), casterId)
					}
					reflected = true
				}
			}
		}
	}

	if reflected {
		// On reflect: monster takes no damage AND no monster status is applied
		// for this entry (FREEZE/STUN/etc. would let the player slip through
		// the reflect's intent).
		return
	}

	if err := deps.applyDamage(f, di.MonsterId(), casterId, damages, byte(ai.AttackType())); err != nil {
		l.WithError(err).Errorf("Unable to apply damage to monster [%d] from character [%d].", di.MonsterId(), casterId)
	}

	// Apply monster status effects from skill (e.g., freeze, poison, stun).
	if len(se.MonsterStatus()) > 0 {
		ms := make(map[string]int32)
		for k, v := range se.MonsterStatus() {
			ms[k] = int32(v)
		}
		if _, isVenom := ms["VENOM"]; isVenom {
			stats := deps.loadVenomStats()
			coef := 0.1 + rand.Float64()*0.1
			ms["VENOM"] = snapshotVenomDamagePerTick(int(stats.Luck), int(stats.MagicAttack), coef)
		}
		_ = deps.applyStatus(f, di.MonsterId(), casterId, uint32(ai.SkillId()), skillLevel, ms, uint32(se.Duration()))
	}
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

						// Skip the generic cost block when a per-skill
						// dispatcher entry exists — that handler owns
						// HP/MP cost (and any cooldown) on the buff-side
						// CharacterUseSkill packet. Without this gate,
						// dual-packet skills like Heal would
						// double-deduct MP.
						if _, registered := handler.Lookup(skill3.Id(ai.SkillId())); !registered {
							if se.HPConsume() > 0 {
								_ = cp.ChangeHP(s.Field(), s.CharacterId(), -int16(se.HPConsume()))
							}
							if se.MPConsume() > 0 {
								_ = cp.ChangeMP(s.Field(), s.CharacterId(), -int16(se.MPConsume()))
							}
							if se.ItemConsume() > 0 {
								if pos, found := findItemSlotInInventory(c.Inventory(), se.ItemConsume()); found {
									_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), charcon.Id(s.CharacterId()), itemconst.Id(se.ItemConsume()), pos, 0)
								} else {
									l.Warnf("Character [%d] cast skill [%d] requiring item [%d] but no such item found in inventory; cast permitted (defense-in-depth gate only).", s.CharacterId(), ai.SkillId(), se.ItemConsume())
								}
							}
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

					// Lazy effective-stats fetch: only needed when a damage entry
					// produces a VENOM apply. Cached for the duration of one attack.
					var venomStats effective_stats.RestModel
					venomStatsLoaded := false
					loadVenomStats := func() effective_stats.RestModel {
						if venomStatsLoaded {
							return venomStats
						}
						venomStatsLoaded = true
						stats, sErr := effective_stats.NewProcessor(l, ctx).GetByCharacterId(s.WorldId(), s.ChannelId(), s.CharacterId())
						if sErr != nil {
							l.WithError(sErr).Errorf("Unable to fetch effective stats for character [%d]; venom DPT will fall back to zero.", s.CharacterId())
							return effective_stats.RestModel{}
						}
						venomStats = stats
						return venomStats
					}

					deps := damageInfoEntryDeps{
						getReflect:        mirror.GetReflect,
						getMonster:        mp.GetById,
						applyDamage:       mp.Damage,
						emitReflectDamage: mp.EmitDamageReflected,
						applyStatus:       mp.ApplyStatus,
						loadVenomStats:    loadVenomStats,
					}
					for _, di := range ai.DamageInfo() {
						processDamageInfoEntry(
							l, di, ai, se, uint32(sk.Level()),
							s.CharacterId(), c.X(), c.Y(),
							s.Field(), t, attackKind,
							deps,
						)
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
