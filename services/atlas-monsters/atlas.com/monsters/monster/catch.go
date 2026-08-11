package monster

import (
	"atlas-monsters/monster/consumable"
	"crypto/rand"
	"math"
	"math/big"
)

// testConsumableLookup is a test-only override for the catch-item data lookup,
// mirroring testInformationLookup. Nil in production.
var testConsumableLookup func(itemId uint32) (consumable.Model, error)

// testCatchRoll is a test-only override for the probability roll. Nil in
// production, where rollCatch uses crypto/rand.
var testCatchRoll func(chance uint32) (bool, error)

// effectiveCatchChance applies bridlePropChg as a ONE-SHOT multiplier on
// bridleProp, clamped to 100 (design assumption A-2). Both values are
// server-side WZ data the client never reads, so no IDB can settle this; a
// per-attempt escalation was rejected because it would need per-(character,
// monster) state nothing else in the codebase keeps. A zero bridleProp means
// the item is deterministic once species and HP pass (FR-3.5).
func effectiveCatchChance(prop uint32, chg float64) uint32 {
	if prop == 0 {
		return 0
	}
	if chg <= 0 {
		return minChance(prop)
	}
	return minChance(uint32(math.Round(float64(prop) * chg)))
}

func minChance(v uint32) uint32 {
	if v > 100 {
		return 100
	}
	return v
}

// rollCatch draws [0,100) from a CSPRNG and reports whether it beat the chance,
// the same shape rollReward uses in atlas-consumables.
func rollCatch(chance uint32) (bool, error) {
	if testCatchRoll != nil {
		return testCatchRoll(chance)
	}
	if chance >= 100 {
		return true, nil
	}
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return false, err
	}
	return uint32(n.Int64()) < chance, nil
}

// catchHpGatePasses reports whether the monster is weak enough. mobHP is a
// PERCENTAGE of max HP (design assumption A-1 — the client never performs this
// check, so it cannot be read from any IDB). The comparison is cross-multiplied
// precisely so integer truncation cannot let a full-HP monster through at
// mobHP < 100: hp <= maxHp * mobHP / 100 becomes hp * 100 <= maxHp * mobHP.
// A zero mobHP means no gate.
func catchHpGatePasses(hp uint32, maxHp uint32, mobHP uint32) bool {
	if mobHP == 0 {
		return true
	}
	return uint64(hp)*100 <= uint64(maxHp)*uint64(mobHP)
}

// Catch resolves a bridle (catch-item) capture attempt. It is fail-closed and
// authoritative: atlas-consumables validated the ITEM, but every monster-state
// check happens here, exactly as Kill re-checks alive+boss rather than trusting
// the caller.
//
// Emission contract:
//   - success: CATCH_RESOLVED(true) -> CAUGHT -> DESTROYED. The economic
//     outcome goes first so it is the first thing attempted after the claim.
//   - a check failed: CATCH_RESOLVED(false, cause) + CATCH_FAILED(cause).
//     Nothing is removed and no KILLED/DESTROYED fires — a catch awards no
//     experience, rolls no drops, and emits no death events (FR-3.6).
//   - monster gone or claim lost: CATCH_RESOLVED(false, UNRESOLVED) +
//     CATCH_FAILED(UNRESOLVED). The resolved event is what cancels the caller's
//     reservation; the channel renders no failure packet for UNRESOLVED, only
//     the unlock. A redelivery is harmless because the caller's once-handler has
//     already deregistered.
//   - data lookup failed: nothing at all (fail-closed).
func (p *ProcessorImpl) Catch(uniqueId uint32, characterId uint32, itemId uint32) {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil || !m.Alive() {
		p.l.Debugf("CATCH: monster [%d] not found or already dead; reporting unresolved for character [%d].", uniqueId, characterId)
		p.emitCatchUnresolved(uniqueId, characterId, itemId)
		return
	}

	var ci consumable.Model
	var ciErr error
	if testConsumableLookup != nil {
		ci, ciErr = testConsumableLookup(itemId)
	} else {
		ci, ciErr = consumable.NewProcessor(p.l, p.ctx).GetById(itemId)
	}
	if ciErr != nil {
		p.l.WithError(ciErr).Errorf("CATCH: catch-item [%d] lookup failed; dropping (fail-closed).", itemId)
		return
	}

	if m.MonsterId() != ci.MonsterId() {
		p.l.Debugf("CATCH: item [%d] targets mob [%d] but monster [%d] is mob [%d].", itemId, ci.MonsterId(), uniqueId, m.MonsterId())
		p.emitCatchFailure(m, characterId, itemId, CatchCauseSpeciesMismatch)
		return
	}
	if !catchHpGatePasses(m.Hp(), m.MaxHp(), ci.MonsterHp()) {
		p.l.Debugf("CATCH: monster [%d] hp [%d]/[%d] above the [%d]%% gate for item [%d].", uniqueId, m.Hp(), m.MaxHp(), ci.MonsterHp(), itemId)
		p.emitCatchFailure(m, characterId, itemId, CatchCauseHpTooHigh)
		return
	}
	if chance := effectiveCatchChance(ci.BridleProp(), ci.BridlePropChg()); chance > 0 {
		won, rerr := rollCatch(chance)
		if rerr != nil {
			p.l.WithError(rerr).Errorf("CATCH: roll failed for item [%d]; dropping (fail-closed).", itemId)
			return
		}
		if !won {
			p.l.Debugf("CATCH: monster [%d] roll failed at [%d]%% for item [%d].", uniqueId, chance, itemId)
			p.emitCatchFailure(m, characterId, itemId, CatchCauseRollFailed)
			return
		}
	}

	claimed, ok, cerr := GetMonsterRegistry().ClaimMonster(p.ctx, p.t, uniqueId)
	if cerr != nil {
		p.l.WithError(cerr).Errorf("CATCH: claim failed for monster [%d]; dropping (fail-closed).", uniqueId)
		return
	}
	if !ok {
		p.l.Debugf("CATCH: monster [%d] claim lost by character [%d].", uniqueId, characterId)
		p.emitCatchUnresolved(uniqueId, characterId, itemId)
		return
	}

	GetDropTimerRegistry().Unregister(p.ctx, p.t, uniqueId)
	GetAttackCooldownRegistry().ClearCooldowns(p.ctx, p.t, uniqueId)

	_ = p.emit(EnvEventTopicMonsterCatch, catchResolvedEventProvider(claimed, characterId, itemId, true, ""))
	_ = p.emit(EnvEventTopicMonsterStatus, caughtStatusEventProvider(claimed, characterId, itemId))
	_ = p.emit(EnvEventTopicMonsterStatus, destroyedStatusEventProvider(claimed))
}

func (p *ProcessorImpl) emitCatchFailure(m Model, characterId uint32, itemId uint32, cause string) {
	_ = p.emit(EnvEventTopicMonsterCatch, catchResolvedEventProvider(m, characterId, itemId, false, cause))
	_ = p.emit(EnvEventTopicMonsterStatus, catchFailedStatusEventProvider(m, characterId, itemId, cause))
}

// emitCatchUnresolved reports a lost race. The monster model is gone, so the
// events carry a bare field-less model built from the uniqueId alone; the
// consumers key on characterId and itemId, not on the field.
func (p *ProcessorImpl) emitCatchUnresolved(uniqueId uint32, characterId uint32, itemId uint32) {
	m := Model{uniqueId: uniqueId}
	p.emitCatchFailure(m, characterId, itemId, CatchCauseUnresolved)
}
