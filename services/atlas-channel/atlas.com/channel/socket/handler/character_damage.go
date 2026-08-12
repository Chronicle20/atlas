package handler

import (
	"atlas-channel/battleship"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	skill2 "atlas-channel/character/skill"
	monsterdata "atlas-channel/data/monster"
	dataskill "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"math"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// damageMitigationDeps injects every lookup and side effect the
// damage-taken pipeline performs, mirroring damageInfoEntryDeps on the
// attack path, so tests drive the pipeline with fakes.
type damageMitigationDeps struct {
	getBuffs           func(characterId uint32) ([]buff.Model, error)
	getSkills          func(characterId uint32) ([]skill2.Model, error)
	getEffect          func(skillId uint32, level byte) (effect.Model, error)
	getMonster         func(monsterId uint32) (monster.Model, error)
	getMonsterTemplate func(templateId uint32) (monsterdata.Model, error)
	changeHP           func(f field.Model, characterId uint32, amount int16) error
	changeMP           func(f field.Model, characterId uint32, amount int16) error
	requestChangeMeso  func(f field.Model, characterId uint32, actorId uint32, actorType string, amount int32) error
	damageMonster      func(f field.Model, monsterId uint32, characterId uint32, damages []uint32, attackType byte) error
	inProtectiveMist   func(f field.Model, characterId uint32, x, y int16) bool
}

func CharacterDamageHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := packetmodel.NewDamageTakenInfo(s.CharacterId())
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			return
		}

		// Foreign-session announce always fires with the client-reported
		// event and is never blocked on mitigation (FR-2.5).
		err = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(charpkt.CharacterDamageWriter)(charpkt.NewCharacterDamage(c.Id(), p.AttackIdx(), p.Damage(), p.MonsterTemplateId(), p.Left()).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce character [%d] has been damaged to foreign characters in map [%d].", s.CharacterId(), s.MapId())
		}

		// Battleship: damage taken while riding drains the ship's parallel
		// HP pool (FR-3.1); the character HP change below is unaffected. A
		// non-breaking drain reports remaining ship HP via the skill-cooldown
		// packet carrying the config-resolved gauge pseudo-skill id
		// (FR-3.4 / DOM-25). Break (dismount + cooldown) is handled inside
		// Drain; the resulting client packets flow through the existing buff
		// and skill consumers.
		//
		// Drains by the client-reported damage, deliberately NOT by the
		// post-mitigation amount that processDamageTaken applies to character
		// HP: the ship pool is a parallel pool, and this is the behavior
		// task-153 specified and verified. Whether Achilles/MagicGuard/etc.
		// should also reduce ship damage is a separate design question — see
		// docs/tasks/task-153-corsair-battleship/backfill.md.
		res := battleship.NewProcessor(l, ctx).Drain(s.Field(), s.CharacterId(), p.Damage())
		if shouldAnnounceGauge(res.Status) {
			announceShipHpGauge(l, ctx, wp, s, res.RemainingHP)
		}

		t := tenant.MustFromContext(ctx)
		cp := character.NewProcessor(l, ctx)
		mp := monster.NewProcessor(l, ctx)
		deps := damageMitigationDeps{
			getBuffs:           buff.NewProcessor(l, ctx).GetByCharacterId,
			getSkills:          skill2.NewProcessor(l, ctx).GetByCharacterId,
			getEffect:          dataskill.NewProcessor(l, ctx).GetEffect,
			getMonster:         mp.GetById,
			getMonsterTemplate: monsterdata.NewProcessor(l, ctx).GetById,
			changeHP:           cp.ChangeHP,
			changeMP:           cp.ChangeMP,
			requestChangeMeso:  cp.RequestChangeMeso,
			damageMonster:      mp.Damage,
			inProtectiveMist:   newSmokeCheck(l, ctx, t),
		}
		processDamageTaken(l, t, s.Field(), p, c, deps)
	}
}

// achillesSkillIdForJob selects the flat-reduction passive by job: the
// client's GetAchillesReduce picks Achilles for jobs 112/122/132 and High
// Defense for Aran (2112), same formula (IDA-verified, all versions).
func achillesSkillIdForJob(jobId job.Id) (skillconst.Id, bool) {
	switch jobId {
	case job.HeroId:
		return skillconst.HeroAchillesId, true
	case job.PaladinId:
		return skillconst.PaladinAchillesId, true
	case job.DarkKnightId:
		return skillconst.DarkKnightAchillesId, true
	case job.AranStage4Id:
		return skillconst.AranStage4HighDefenseId, true
	}
	return 0, false
}

// buffAmounts extracts the roster statup values from active buffs.
type buffAmounts struct {
	magicGuard          int32
	infinity            bool
	powerGuard          int32
	mesoGuard           int32
	manaReflect         bool
	manaReflectSourceId int32
	manaReflectLevel    byte
	comboBarrier        int32
	magicShield         int32
	guard               bool
}

func extractBuffAmounts(buffs []buff.Model) buffAmounts {
	var a buffAmounts
	for _, b := range buffs {
		if b.Expired() {
			continue
		}
		for _, ch := range b.Changes() {
			switch ch.Type() {
			case string(charconst.TemporaryStatTypeMagicGuard):
				a.magicGuard = ch.Amount()
			case string(charconst.TemporaryStatTypeInfinity):
				a.infinity = true
			case string(charconst.TemporaryStatTypePowerGuard):
				a.powerGuard = ch.Amount()
			case string(charconst.TemporaryStatTypeMesoGuard):
				a.mesoGuard = ch.Amount()
			case string(charconst.TemporaryStatTypeManaReflection):
				a.manaReflect = true
				a.manaReflectSourceId = b.SourceId()
				a.manaReflectLevel = b.Level()
			case string(charconst.TemporaryStatTypeComboBarrier):
				a.comboBarrier = ch.Amount()
			case string(charconst.TemporaryStatTypeMagicShield):
				a.magicShield = ch.Amount()
			case string(charconst.TemporaryStatTypeGuard):
				a.guard = true
			}
		}
	}
	return a
}

// processDamageTaken applies the server-authoritative mitigation chain to
// one damage-taken event and emits the resulting deltas. The client's
// damage value is raw pre-mitigation input (IDA-verified); reflect
// amounts are always server-computed (FR-10.3).
func processDamageTaken(
	l logrus.FieldLogger,
	t tenant.Model,
	f field.Model,
	p packetmodel.DamageTakenInfo,
	c character.Model,
	deps damageMitigationDeps,
) {
	characterId := c.Id()

	// Smokescreen: a character standing in a protection mist owned by
	// themselves or an online party member takes nothing at all. This is a
	// SHORT-CIRCUIT, not another mitigation term, because that is what the
	// client does: CUserLocal::SetDamaged jumps to the function epilogue on a
	// positive IsSmokeAreaByPoint (v95 SetDamaged+0x1ef -> loc_93651F),
	// before the miss roll, Power Guard, Meso Guard, Achilles and Magic
	// Guard, and before the damage packet is built. Returning here is what
	// keeps reflect and Meso Guard amounts from being computed off damage the
	// shield zeroed (FR-4.5).
	//
	// Server-authoritative: the position comes from the character model, the
	// rectangle from the mist event, the party from the party service. An
	// honest client in smoke sends nothing, so this exists to stop a crafted
	// one claiming damage it did not take.
	if deps.inProtectiveMist != nil && deps.inProtectiveMist(f, characterId, c.X(), c.Y()) {
		l.Debugf("Character [%d] shielded by a protection mist in map [%d]; damage [%d] dropped.", characterId, f.MapId(), p.Damage())
		return
	}

	// Block sentinel: the client sends damage == -1 for a fully blocked
	// hit (Guardian, Fake/Shadow Shifter, GUARD, v95 Mechanic Perfect
	// Armor) and applies zero HP loss. The old handler applied +1 HP.
	if p.Damage() == -1 {
		// DarkKnightId (132) is intentionally excluded here: it has no
		// block skill (Achilles only, no Guardian/Shifter-style block).
		plausible := p.Guard() ||
			c.JobId() == job.HeroId || c.JobId() == job.PaladinId ||
			c.JobId() == job.NightLordId || c.JobId() == job.ShadowerId
		if !plausible {
			if buffs, err := deps.getBuffs(characterId); err == nil {
				plausible = extractBuffAmounts(buffs).guard
			}
		}
		if !plausible {
			l.Warnf("Character [%d] in map [%d] sent a block sentinel with no plausible block source (job [%d], mob template [%d]). Ignoring damage.", characterId, f.MapId(), c.JobId(), p.MonsterTemplateId())
		}
		return
	}

	raw, adjusted := clampDamage(p.Damage())
	if adjusted {
		l.Warnf("Character [%d] in map [%d] sent out-of-bounds damage [%d] (mob template [%d], attackIdx [%d]). Clamped to [%d].", characterId, f.MapId(), p.Damage(), p.MonsterTemplateId(), p.AttackIdx(), raw)
	}

	mobSourced := p.AttackIdx() >= packetmodel.DamageTypePhysical

	var a buffAmounts
	buffs, err := deps.getBuffs(characterId)
	if err != nil {
		// Buff lookup failure must not leave the hit unapplied: fall back
		// to the unmitigated path (FR-2.4 behavior).
		l.WithError(err).Warnf("Unable to look up buffs for character [%d]; applying unmitigated damage.", characterId)
	} else {
		a = extractBuffAmounts(buffs)
	}

	// Cross-check the client's Power Guard claim against server state
	// (FR-5.4): amounts are never taken from the wire.
	powerGuardSignal := false
	if p.HasReflectExtension() && p.IsPowerGuard() && mobSourced {
		if a.powerGuard <= 0 {
			l.Warnf("Character [%d] claimed Power Guard without an active POWER_GUARD buff (mob [%d], map [%d]). Ignoring claim.", characterId, p.MonsterId(), f.MapId())
		} else if p.ReflectTargetMobId() != p.MonsterId() {
			l.Warnf("Character [%d] Power Guard reflect target [%d] is not the attacking mob [%d] (map [%d]). Ignoring claim.", characterId, p.ReflectTargetMobId(), p.MonsterId(), f.MapId())
		} else {
			powerGuardSignal = true
		}
	}

	// Mana Reflection: the client rolls prop and signals the outcome via
	// a reflect echo without isPowerGuard on a mob skill attack
	// (attackIdx >= 0). Honor the validated signal, recompute the amount.
	manaReflectSignal := false
	var manaReflectPct int32
	if p.HasReflectExtension() && !p.IsPowerGuard() && p.Reflect() > 0 && p.AttackIdx() >= packetmodel.DamageTypeMagic {
		if !a.manaReflect {
			l.Warnf("Character [%d] signaled Mana Reflection without an active MANA_REFLECTION buff (mob [%d], map [%d]). Ignoring claim.", characterId, p.MonsterId(), f.MapId())
		} else {
			eff, effErr := deps.getEffect(uint32(a.manaReflectSourceId), a.manaReflectLevel)
			if effErr != nil {
				l.WithError(effErr).Warnf("Unable to load Mana Reflection effect [%d] level [%d] for character [%d]. Dropping reflect.", a.manaReflectSourceId, a.manaReflectLevel, characterId)
			} else {
				manaReflectSignal = true
				manaReflectPct = int32(eff.X())
			}
		}
	}

	// Warrior/Aran flat-reduction passive: only fetch skills for the jobs
	// that have one (design §5 step 3).
	var achillesPermille int32
	if skillId, ok := achillesSkillIdForJob(c.JobId()); ok {
		skills, sErr := deps.getSkills(characterId)
		if sErr != nil {
			l.WithError(sErr).Warnf("Unable to look up skills for character [%d]; skipping passive reduction.", characterId)
		} else if level := skill2.GetLevel(skills, skillId); level > 0 {
			eff, effErr := deps.getEffect(uint32(skillId), level)
			if effErr != nil {
				l.WithError(effErr).Warnf("Unable to load passive effect [%d] level [%d] for character [%d]; skipping passive reduction.", skillId, level, characterId)
			} else {
				achillesPermille = int32(eff.X())
			}
		}
	}

	// Mob data is only needed when a reflect will actually be computed.
	var mob mobInfo
	if (powerGuardSignal && a.powerGuard > 0) || manaReflectSignal {
		live, mErr := deps.getMonster(p.MonsterId())
		if mErr != nil {
			l.WithError(mErr).Debugf("Reflect target mob [%d] not found for character [%d]; dropping reflect, keeping mitigation.", p.MonsterId(), characterId)
		} else {
			mob.present = true
			mob.alive = live.Hp() > 0
			mob.maxHp = live.MaxHp()
			tmpl, tErr := deps.getMonsterTemplate(p.MonsterTemplateId())
			if tErr != nil {
				l.WithError(tErr).Debugf("Monster template [%d] not found; boss/fixedDamage caps default to non-boss/none.", p.MonsterTemplateId())
			} else {
				mob.boss = tmpl.Boss()
				mob.fixedDamage = tmpl.FixedDamage()
			}
		}
	}

	pgCapDivisor := int32(10)
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		pgCapDivisor = 2
	}
	in := mitigationInput{
		attackIdx:                  p.AttackIdx(),
		rawDamage:                  raw,
		mobSourced:                 mobSourced,
		powerGuardSignal:           powerGuardSignal,
		manaReflectSignal:          manaReflectSignal,
		currentMP:                  c.Mp(),
		meso:                       c.Meso(),
		magicGuardPct:              a.magicGuard,
		infinity:                   a.infinity,
		powerGuardPct:              a.powerGuard,
		mesoGuardPct:               a.mesoGuard,
		comboBarrierPermille:       a.comboBarrier,
		magicShieldPct:             a.magicShield,
		achillesPermille:           achillesPermille,
		manaReflectPct:             manaReflectPct,
		magicShieldOnReducedDamage: t.MajorVersion() >= 87,
		pgCapDivisor:               pgCapDivisor,
		pgFixedDamageOverride:      (t.Region() == "GMS" && t.MajorVersion() >= 95) || t.Region() == "JMS",
	}

	result := computeMitigation(in, mob)
	l.Debugf("Character [%d] damage [%d] mitigated to hp [%d] mp [%d] meso [%d] reflect [%d] (achilles [%d], comboBarrier [%d], magicShield [%d], magicGuard [%d], mesoGuard [%d], powerGuard [%d]).",
		characterId, raw, result.hpLoss, result.mpLoss, result.mesoCost, result.reflect.amount,
		result.breakdown.achillesReduce, result.breakdown.comboBarrierReduce, result.breakdown.magicShieldReduce,
		result.breakdown.magicGuardAbsorbed, result.breakdown.mesoGuarded, result.breakdown.powerGuardReflect)

	_ = deps.changeHP(f, characterId, -clampInt16(result.hpLoss))
	if result.mpLoss > 0 {
		_ = deps.changeMP(f, characterId, -clampInt16(result.mpLoss))
	}
	if result.mesoCost > 0 {
		_ = deps.requestChangeMeso(f, characterId, characterId, "SKILL", -result.mesoCost)
	}
	if result.reflect.amount > 0 {
		_ = deps.damageMonster(f, p.MonsterId(), characterId, []uint32{result.reflect.amount}, result.reflect.attackType)
	}
}

// shouldAnnounceGauge is the call-site gate isolated as a pure predicate so
// it is directly unit-testable: the full handler can't be driven end-to-end
// in this package's tests (the earlier, pre-existing, unseamed
// character.NewProcessor(...).GetById() call returns early without a live
// character service), so the gate itself — the only thing standing between
// a correct and an incorrect announce — is verified here against every
// battleship.DrainStatus value instead. Only DrainDrained carries a valid
// RemainingHP to report; DrainBroke's dismount+cooldown already flows
// through the existing buff/skill consumers, and DrainSkipped/DrainNotRiding
// have no HP change to report at all.
func shouldAnnounceGauge(status battleship.DrainStatus) bool {
	return status == battleship.DrainDrained
}

// announceShipHpGauge sends the client's ship HP gauge: the skill-cooldown
// packet with the config-resolved battleship gauge pseudo-skill id and the
// remaining ship HP as the cooldown value (verified client behavior on
// v83/v84/v87/v95/jms185 — design §1.1). On any resolve miss the packet is
// skipped entirely (fail-loud, never send a guessed wire value).
func announceShipHpGauge(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, remaining int32) {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), charpkt.CharacterSkillCooldownWriter)
	if !ok {
		l.Errorf("Writer options for [%s] missing; battleship HP gauge not sent.", charpkt.CharacterSkillCooldownWriter)
		return
	}
	gaugeId, ok := atlaspacket.ResolveValue(l, opts, "skills", "BATTLESHIP_HP_GAUGE")
	if !ok {
		return
	}
	if err := session.Announce(l)(ctx)(wp)(charpkt.CharacterSkillCooldownWriter)(charpkt.NewCharacterSkillCooldown(gaugeId, gaugeCooldownValue(remaining)).Encode)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce battleship HP gauge to character [%d].", s.CharacterId())
	}
}

// gaugeCooldownValue clamps remaining ship HP into the packet's uint16
// field. Battleship is maxLevel 10 on every version (R-5), so the ceiling is
// the v87+ arm at SLV 10 / charLevel 200 = 29 000 — well inside uint16. The
// clamp is purely defensive.
func gaugeCooldownValue(remaining int32) uint16 {
	if remaining < 0 {
		return 0
	}
	if remaining > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(remaining)
}
