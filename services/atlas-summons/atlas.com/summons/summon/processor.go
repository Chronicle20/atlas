package summon

import (
	skilldata "atlas-summons/data/skill"
	"atlas-summons/data/skill/effect"
	"atlas-summons/effectivestats"
	"atlas-summons/inventory"
	"atlas-summons/kafka/producer"
	monstermsg "atlas-summons/monster"
	"context"
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	summonconst "github.com/Chronicle20/atlas/libs/atlas-constants/summon"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(id uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)
	Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, skillLevel byte, x int16, y int16) (Model, error)
	Move(id uint32, senderCharacterId uint32, x int16, y int16, stance byte, rawMovement []byte) error
	Attack(id uint32, senderCharacterId uint32, direction byte, targets []AttackTarget) error
	Damage(id uint32, senderCharacterId uint32, amount int32, monsterIdFrom uint32) error
	Despawn(id uint32, animated bool) error
	DespawnAllForOwner(ownerCharacterId uint32) error
}

// AttackTarget is one {monster, reported damage} pair from a summon-attack packet.
type AttackTarget struct {
	MonsterId uint32
	Damage    uint32
}

// emitter publishes a kafka message provider to a topic. ProcessorImpl uses this
// indirection so later phases can intercept event emissions in tests without
// spinning up kafka. Production wiring uses producer.ProviderImpl.
type emitter func(topic string, provider model.Provider[[]kafka.Message]) error

// effectSource provides per-skill effect data. The default implementation is
// the data/skill REST processor; tests substitute a stub so spawn logic is
// unit-testable without a live atlas-data.
type effectSource interface {
	GetEffect(skillId uint32, level byte) (effect.Model, error)
}

// statsSource provides a character's session-effective combat stats. The default
// implementation is the effectivestats REST processor; tests substitute a stub so
// the damage ceiling is unit-testable without a live atlas-effective-stats.
type statsSource interface {
	GetByCharacter(worldId world.Id, channelId channel.Id, characterId uint32) (effectivestats.Model, error)
}

// weaponSource resolves the owner's equipped main-weapon type, required by the
// weapon-type-aware physical damage ceiling (FaithfulMaxPerHit). The default
// implementation is the inventory REST processor; tests substitute a stub.
type weaponSource interface {
	GetEquippedWeaponType(characterId uint32) (item.WeaponType, error)
}

type ProcessorImpl struct {
	l       logrus.FieldLogger
	ctx     context.Context
	t       tenant.Model
	emit    emitter
	effects effectSource
	stats   statsSource
	equip   weaponSource
	// proc decides whether a prop-gated status effect lands. The default is a real
	// RNG roll (see rollProc); tests inject a deterministic function to force or
	// suppress procs.
	proc func(prop float64) bool
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l: l, ctx: ctx, t: tenant.MustFromContext(ctx),
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			return producer.ProviderImpl(l)(ctx)(topic)(provider)
		},
		effects: skilldata.NewProcessor(l, ctx),
		stats:   effectivestats.NewProcessor(l, ctx),
		equip:   inventory.NewProcessor(l, ctx),
		proc:    rollProc,
	}
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, p.t, id)
}
func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return GetRegistry().GetInField(p.ctx, p.t, f)
}

// Spawn classifies the cast skill against the summon roster, removes any
// same-skill or mobility-conflicting existing summon for the owner, fetches the
// skill effect for HP/duration, persists the new summon, and emits CREATED.
// A non-summon skill is a graceful no-op (FR-1.3).
func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, skillLevel byte, x int16, y int16) (Model, error) {
	entry, ok := summonconst.Lookup(skillId)
	if !ok {
		p.l.Debugf("Skill [%d] is not a summon; no spawn.", skillId) // FR-1.3 graceful no-op
		return Model{}, nil
	}

	// FR-2.4 / FR-2.5: remove same-skill instance and conflicting-mobility-class instance.
	existing, _ := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	for _, e := range existing {
		if e.SkillId() == skillId || conflictsMobility(entry.Movement, e.MovementType()) {
			_ = p.Despawn(e.Id(), false)
		}
	}

	eff, err := p.effects.GetEffect(skillId, skillLevel)
	if err != nil {
		p.l.WithError(err).Warnf("No effect data for summon skill [%d]; aborting spawn.", skillId)
		return Model{}, err
	}

	id := GetIdAllocator().Allocate(p.ctx, p.t)
	now := time.Now()
	expires := now.Add(time.Duration(eff.Duration()) * time.Millisecond)

	hp := int32(0)
	if entry.Type == summonconst.TypePuppet {
		hp = int32(eff.X())
	} else if entry.Type == summonconst.TypeBuffAura {
		hp = int32(eff.X()) + 1 // Cosmic Beholder hp = x + 1
	}

	b := NewBuilder().
		SetId(id).SetOwnerCharacterId(ownerCharacterId).SetSkillId(skillId).SetSkillLevel(skillLevel).
		SetSummonType(SummonType(entry.Type)).SetMovementType(MovementType(entry.Movement)).
		SetField(f).SetX(x).SetY(y).SetHp(hp).SetMaxHp(hp).
		SetSpawnTime(now).SetExpiresAt(expires).SetAnimated(true)

	// Phase 5: Beholder aura snapshot (next-heal/next-buff timers, buff changes)
	// is layered onto this builder.
	m := b.Build()

	if err := GetRegistry().Put(p.ctx, p.t, m); err != nil {
		GetIdAllocator().Release(p.ctx, p.t, id)
		return Model{}, err
	}
	if err := p.emit(EnvEventTopicSummonStatus, createdEventProvider(m)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit CREATED for summon [%d].", id)
	}
	// Phase 4: emit ADD_PUPPET to atlas-monsters for puppet summons.
	// Phase 5: Beholder timer init.
	return m, nil
}

// Move relays an owner's summon-move packet: it verifies ownership (a character
// may only move a summon it owns — §11), updates the persisted position, and
// emits MOVED carrying the raw movement blob for byte-faithful rebroadcast. A
// missing summon or a non-owner sender is a graceful no-op (returns nil).
func (p *ProcessorImpl) Move(id uint32, senderCharacterId uint32, x int16, y int16, stance byte, rawMovement []byte) error {
	m, err := GetRegistry().Get(p.ctx, p.t, id)
	if err != nil {
		return nil
	}
	if m.OwnerCharacterId() != senderCharacterId {
		p.l.Infof("Character [%d] moved summon [%d] it does not own; dropping.", senderCharacterId, id) // §11 ownership
		return nil
	}
	updated, err := GetRegistry().Update(p.ctx, p.t, id, func(cur Model) Model {
		return cur.Move(x, y, stance)
	})
	if err != nil {
		return err
	}
	return p.emit(EnvEventTopicSummonStatus, movedEventProvider(updated, rawMovement))
}

// Attack relays an owner's summon-attack packet. It verifies ownership, then for
// each reported target it credits the OWNER with the (server-clamped) damage via a
// monster DAMAGE command (FR-4.2 — so XP/drops/kill credit accrue to the player,
// not the summon), applies stun/freeze where the roster + proc allow (FR-4.4),
// and emits an ATTACKED event carrying the clamped targets for rebroadcast.
// Gaviota self-cancels after a single attack (FR-4.5). A missing summon or a
// non-owner sender is a graceful no-op (returns nil).
func (p *ProcessorImpl) Attack(id uint32, senderCharacterId uint32, direction byte, targets []AttackTarget) error {
	m, err := GetRegistry().Get(p.ctx, p.t, id)
	if err != nil {
		return nil // already gone
	}
	if m.OwnerCharacterId() != senderCharacterId {
		p.l.Infof("Character [%d] attacked with summon [%d] it does not own; dropping.", senderCharacterId, id) // §11 ownership
		return nil
	}

	eff, err := p.effects.GetEffect(m.SkillId(), m.SkillLevel())
	if err != nil {
		return err
	}

	// Owner combat stats drive the per-hit ceiling (FR-4.3), a faithful port of
	// Cosmic's weapon-type-aware calcMaxDamage. If stats are unavailable, set
	// max=0 so clampDamage treats it as "no ceiling" — never zero legit damage.
	var max int64
	stats, serr := p.stats.GetByCharacter(m.Field().WorldId(), m.Field().ChannelId(), m.OwnerCharacterId())
	if serr != nil {
		p.l.WithError(serr).Warnf("No effective-stats for owner [%d]; summon [%d] damage not clamped this hit.", m.OwnerCharacterId(), id)
		max = 0
	} else {
		magic := eff.WeaponAttack() == 0
		// The physical branch needs the equipped weapon type. A failed lookup
		// degrades to WeaponTypeNone (Cosmic's SWORD1H no-weapon fallback) rather
		// than disabling the clamp; magic ignores weapon type entirely.
		weaponType := item.WeaponTypeNone
		if !magic {
			if wt, werr := p.equip.GetEquippedWeaponType(m.OwnerCharacterId()); werr != nil {
				p.l.WithError(werr).Warnf("No equipped-weapon type for owner [%d]; summon [%d] physical ceiling uses SWORD1H fallback.", m.OwnerCharacterId(), id)
			} else {
				weaponType = wt
			}
		}
		max = FaithfulMaxPerHit(magic, stats.WeaponAttack(), stats.MagicAttack(), stats.Intelligence(),
			stats.Strength(), stats.Dexterity(), stats.Luck(), weaponType, eff.WeaponAttack(), eff.MagicAttack())
	}

	statuses := monsterStatusFor(m.SkillId(), eff)

	clampedTargets := make([]AttackTarget, 0, len(targets))
	for _, tgt := range targets {
		dmg := clampDamage(tgt.Damage, max)
		if max > 0 && int64(tgt.Damage) > max {
			// FR-4.3 alert: warn-only (clamp-and-continue). Intentionally does NOT
			// emit to COMMAND_TOPIC_BAN (context.md §9 — false-positive risk).
			p.l.Infof("Summon [%d] owner [%d] reported damage [%d] > ceiling [%d] on mob [%d]; clamped. (FR-4.3 alert)",
				id, m.OwnerCharacterId(), tgt.Damage, max, tgt.MonsterId)
		}
		clampedTargets = append(clampedTargets, AttackTarget{MonsterId: tgt.MonsterId, Damage: dmg})

		// FR-4.2: credit the owner via monster DAMAGE.
		if err := p.emit(monstermsg.EnvCommandTopic, monstermsg.MonsterDamageProvider(m.Field(), tgt.MonsterId, m.OwnerCharacterId(), []uint32{dmg})); err != nil {
			p.l.WithError(err).Errorf("Unable to emit monster DAMAGE for summon [%d] target [%d].", id, tgt.MonsterId)
		}

		// FR-4.4: stun/freeze, gated by the skill's prop chance.
		if len(statuses) > 0 && p.proc(eff.Prop()) {
			if err := p.emit(monstermsg.EnvCommandTopic, monstermsg.MonsterApplyStatusProvider(m.Field(), tgt.MonsterId, m.OwnerCharacterId(), m.SkillId(), m.SkillLevel(), eff, statuses)); err != nil {
				p.l.WithError(err).Errorf("Unable to emit monster APPLY_STATUS for summon [%d] target [%d].", id, tgt.MonsterId)
			}
		}
	}

	if err := p.emit(EnvEventTopicSummonStatus, attackedEventProvider(m, direction, clampedTargets)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit ATTACKED for summon [%d].", id)
	}

	// FR-4.5: Gaviota self-cancels after one attack.
	if e, ok := summonconst.Lookup(m.SkillId()); ok && e.OneShot {
		_ = p.Despawn(id, true)
	}
	return nil
}

// monsterStatusFor returns the monster status map (STUN/FREEZE) a summon applies
// on hit, driven by the roster flags and the skill effect's own monsterStatus
// map. Values are the status level (1) for boolean roster flags; effect-supplied
// statuses carry their configured level. An empty map means no status applies.
func monsterStatusFor(skillId uint32, eff effect.Model) map[string]int32 {
	statuses := make(map[string]int32)
	if e, ok := summonconst.Lookup(skillId); ok {
		if e.Stun {
			statuses["STUN"] = 1
		}
		if e.Freeze {
			statuses["FREEZE"] = 1
		}
	}
	// Layer any effect-declared statuses (e.g. data-driven freeze level).
	for k, v := range eff.MonsterStatus() {
		statuses[k] = int32(v)
	}
	return statuses
}

// rollProc is the default proc decision: prop is the skill's 0.0-1.0 chance. A
// prop >= 1.0 always procs; otherwise a uniform random draw gates it. Tests
// override ProcessorImpl.proc to make this deterministic.
func rollProc(prop float64) bool {
	if prop >= 1.0 {
		return true
	}
	if prop <= 0.0 {
		// Treat a missing/zero prop as always-apply: a roster status flag with no
		// configured chance should still land (Cosmic applies these unconditionally).
		return true
	}
	return rand.Float64() < prop
}

// Damage is implemented in Phase 4 (summon takes damage).
func (p *ProcessorImpl) Damage(id uint32, senderCharacterId uint32, amount int32, monsterIdFrom uint32) error {
	return nil
}

// Despawn removes a summon from the registry, releases its oid, and emits
// DESTROYED. A missing summon is treated as already gone (no error).
func (p *ProcessorImpl) Despawn(id uint32, animated bool) error {
	m, err := GetRegistry().Get(p.ctx, p.t, id)
	if err != nil {
		return nil // already gone
	}
	if err := GetRegistry().Remove(p.ctx, p.t, id); err != nil {
		return err
	}
	GetIdAllocator().Release(p.ctx, p.t, id)
	if err := p.emit(EnvEventTopicSummonStatus, destroyedEventProvider(m, animated)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit DESTROYED for summon [%d].", id)
	}
	// Phase 4: emit REMOVE_PUPPET to atlas-monsters for puppet summons.
	// Phase 5: Beholder timer cleanup is implicit (registry removal).
	return nil
}

// DespawnAllForOwner removes every summon owned by the character. Used by the
// logout / channel-change / map-change cascade.
func (p *ProcessorImpl) DespawnAllForOwner(ownerCharacterId uint32) error {
	ms, err := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	if err != nil {
		return err
	}
	for _, m := range ms {
		_ = p.Despawn(m.Id(), false)
	}
	return nil
}

// conflictsMobility implements Cosmic StatEffect.java:1024-1029: a new stationary
// summon cancels the existing stationary one; a new non-stationary cancels the
// existing non-stationary one.
func conflictsMobility(newMove summonconst.Movement, existing MovementType) bool {
	newStationary := newMove == summonconst.MovementStationary
	existingStationary := existing == MovementStationary
	return newStationary == existingStationary
}
