package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	buff2 "atlas-channel/kafka/message/buff"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	// energyChargeGainPerMob is the bar gain per attacked monster. Cosmic
	// calls handleEnergyChargeGain once per attacked mob
	// (CloseRangeDamageHandler.java:136-140); Atlas collapses the loop into
	// one emit of 102 x mobs so the attack path costs at most one Kafka
	// message (NFR-1).
	energyChargeGainPerMob = int32(102)
	// energyChargeCap is the accumulation ceiling; energyChargedValue is the
	// charged-state SENTINEL, not a bar reading — nothing may treat it as
	// "150% full" (FR-3.1). Both are shared, so the accumulation ceiling here
	// cannot drift from the promotion in the buff consumer or the stat payoff
	// in atlas-effective-stats.
	energyChargeCap    = constants.EnergyChargeCap
	energyChargedValue = constants.EnergyChargedValue
)

// energyLine is the Energy Charge skill line a character owns: the tenant's
// wire id for that line's Energy Charge skill, plus the character's level in
// it. The level selects the WZ effect row that supplies the charged window's
// duration and its weapon-attack payoff.
type energyLine struct {
	skillId skill3.Id
	level   byte
}

// energyChargeLine resolves the character's Energy Charge line from owned
// skills, adventurer branch first. ok == false when the character owns
// neither variant at level > 0.
//
// Identities, not raw ids: set.Wire returns false for the Cygnus identity on
// gms_v61 (Thunder Breaker postdates it), so that branch degrades to a no-op
// rather than resolving a bogus id (AC-10). No job check is needed —
// owning the skill at level > 0 already implies the line, which is all
// Cosmic's isCygnus() split was ever deciding.
func energyChargeLine(set skill3.Set, skills []skill.Model) (energyLine, bool) {
	find := func(id skill3.Id) byte {
		for _, s := range skills {
			if s.Id() == id {
				return s.Level()
			}
		}
		return 0
	}
	for _, identity := range []skill3.Identity{skill3.MarauderEnergyCharge, skill3.ThunderBreakerStage2EnergyCharge} {
		wire, ok := set.Wire(identity)
		if !ok {
			continue
		}
		if lvl := find(wire); lvl > 0 {
			return energyLine{skillId: wire, level: lvl}, true
		}
	}
	return energyLine{}, false
}

// energyChargeGainAmount is the bar gain for one attack: 102 per attacked
// monster, or nothing when the attack hit nothing (FR-2.4).
func energyChargeGainAmount(mobsHit int) int32 {
	if mobsHit <= 0 {
		return 0
	}
	return energyChargeGainPerMob * int32(mobsHit)
}

// energyChargeQualifies reports whether an attack feeds the energy bar. Melee
// covers every close-range attack including the basic attack (skillId 0);
// AttackTypeEnergy is the Energy Charge aura's own touch damage, which Cosmic
// routes through the same close-range handler; on the ranged path ONLY
// Thunder Breaker Shark Wave qualifies (RangedAttackHandler.java:90-99).
func energyChargeQualifies(at packetmodel.AttackType, attackId skill3.Identity, attackIdOk bool) bool {
	switch at {
	case packetmodel.AttackTypeMelee, packetmodel.AttackTypeEnergy:
		return true
	case packetmodel.AttackTypeRanged:
		return attackIdOk && skill3.IsIdentity(attackId, skill3.ThunderBreakerStage3SharkWave)
	}
	return false
}

// isEnergyBlast reports whether a cast is the charge-gated Energy Blast.
// Energy Blast is the only skill in the family that carries no mpCon at any
// level — the sole WZ evidence that it is energy-costed rather than
// MP-costed. Shockwave (mpCon 18) and Shark Wave (mpCon 15) are NOT gated
// (design.md OQ-3).
func isEnergyBlast(attackId skill3.Identity, attackIdOk bool) bool {
	return attackIdOk && skill3.IsIdentity(attackId, skill3.MarauderEnergyBlast, skill3.ThunderBreakerStage2EnergyBlast)
}

// energyChargeDeps groups the side-effecting call energyChargeTryUpdate makes
// so tests can drive every branch without a real processor or Kafka producer.
type energyChargeDeps struct {
	emitUpsert func(sourceId int32, level byte, amount int32, capValue int32) error
}

// energyChargeProductionDeps wires energyChargeDeps to the buff
// UPDATE_STAT_VALUE emitter for one attack. CreateIfMissing is what keeps the
// attack path free of a REST read: the channel never has to ask whether the
// bar buff exists (design.md §4.2).
func energyChargeProductionDeps(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32) energyChargeDeps {
	bp := buff.NewProcessor(l, ctx)
	return energyChargeDeps{
		emitUpsert: func(sourceId int32, level byte, amount int32, capValue int32) error {
			return bp.UpdateStatValue(f, characterId, buff.StatValueUpdate{
				SourceId:        sourceId,
				StatType:        string(constants.TemporaryStatTypeEnergyCharge),
				Operation:       buff2.StatOperationIncrement,
				Amount:          amount,
				Cap:             capValue,
				CreateIfMissing: true,
				Level:           level,
			})
		},
	}
}

// energyChargeTryUpdate applies Energy Charge bar bookkeeping for one
// qualifying attack: at most ONE emit, carrying 102 x mobs clamped to 10000.
//
// "No gain while charged" (FR-2.5) is structural rather than a guard here:
// at the 15000 sentinel atlas-buffs' current >= cap test makes the increment
// a no-op and emits no status event, so the channel broadcasts nothing.
//
// All failures are logged and swallowed — the attack pipeline never fails on
// energy bookkeeping (FR-2.6 / NFR-2).
func energyChargeTryUpdate(l logrus.FieldLogger, set skill3.Set, c character.Model, ai packetmodel.AttackInfo, deps energyChargeDeps) {
	line, ok := energyChargeLine(set, c.Skills())
	if !ok {
		return
	}
	amount := energyChargeGainAmount(len(ai.DamageInfo()))
	if amount == 0 {
		return
	}
	if err := deps.emitUpsert(int32(line.skillId), line.level, amount, energyChargeCap); err != nil {
		l.WithError(err).Errorf("Energy Charge: gain emit failed for character [%d] energy line [%d].", c.Id(), line.skillId)
	}
}

// energyBlastPermitted gates Energy Blast on the caster being charged. Energy
// Blast is an ATTACK skill (WZ: damage/mobCount/lt/rb, no time), so it never
// reaches CharacterUseSkillHandleFunc — the gate belongs beside
// battleshipAttackPermitted in processAttack, and the rejection stays soft
// (return false, never destroy the session).
//
// Reads the pod-local mirror: zero I/O on the permitted path. Returns the
// mirrored bar alongside the verdict so the caller can log it and re-announce.
//
// Fails OPEN on a missing mirror entry. A miss means "unknown" — a fresh
// channel or a restarted pod, not an empty bar — and an unknown must never eat
// a legitimate cast. A KNOWN zero, by contrast, is a real reading and is
// rejected.
//
// This is a deliberate divergence from Cosmic, which performs no server-side
// charge check at all; no client-side gate was found in the v83 IDB either
// (design.md OQ-3). The fail-open plus the re-announce below bound the damage
// to "one cast allowed that Cosmic would also have allowed".
func energyBlastPermitted(t tenant.Model, characterId uint32, attackId skill3.Identity, attackIdOk bool) (bool, int32) {
	if !isEnergyBlast(attackId, attackIdOk) {
		return true, 0
	}
	v, ok := buff.GetEnergyMirror().Get(t, characterId)
	if !ok {
		return true, 0
	}
	return v == energyChargedValue, v
}

// energyReannounceAuthoritative re-sends the caster's true ENERGY_CHARGE bar
// after a rejected Energy Blast, so a client whose bar drifted (a dropped
// STAT_UPDATED, a reconnect before the buff replayed) resynchronises instead
// of losing the skill with no feedback (design.md OQ-1 resolution (b)).
//
// This is the ONE REST call the gate is allowed, and only on a rejection —
// the permitted path stays I/O-free. Failures are logged and swallowed: the
// rejection itself already happened.
func energyReannounceAuthoritative(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) {
	bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
	if err != nil {
		l.WithError(err).Errorf("Energy Charge: unable to read authoritative bar for character [%d] after a rejected cast.", s.CharacterId())
		return
	}
	t := tenant.MustFromContext(ctx)
	for _, b := range bs {
		for _, c := range b.Changes() {
			if c.Type() != string(constants.TemporaryStatTypeEnergyCharge) {
				continue
			}
			buff.GetEnergyMirror().Set(t, s.CharacterId(), c.Amount())
			if aerr := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveWriter)(writer.CharacterBuffGiveBody([]buff.Model{b}))(s); aerr != nil {
				l.WithError(aerr).Errorf("Energy Charge: bar re-announce failed for character [%d].", s.CharacterId())
			}
			return
		}
	}
	// No ENERGY_CHARGE buff upstream at all: the mirror was stale. Clear it so
	// the next cast fails open rather than being rejected forever.
	buff.GetEnergyMirror().Clear(t, s.CharacterId())
}
