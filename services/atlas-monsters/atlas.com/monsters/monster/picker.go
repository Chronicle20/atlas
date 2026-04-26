package monster

import (
	"atlas-monsters/monster/information"
	"atlas-monsters/monster/mobskill"
	"context"
	"math/rand"
	"time"

	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Decision is the picker's chosen next skill (or sentinel zero for "no
// skill"). Only the byte-wide skillId/skillLevel travel back into the
// MoveMonsterAck; the millis fields are picker bookkeeping consumed by the
// sweep task.
type Decision struct {
	SkillId                byte
	SkillLevel             byte
	DecidedAtMs            int64
	NextEligibleRepickAtMs int64
}

// IsSentinel reports whether the decision is the "no skill" sentinel. The
// SkillId == 0 check matches PRD §5.1.
func (d Decision) IsSentinel() bool { return d.SkillId == 0 }

// RepickReason names the trigger that caused the picker to run. Used in
// debug/info logs to make production "monster never casts" complaints easy
// to debug.
type RepickReason string

const (
	RepickReasonSpawn         RepickReason = "spawn"
	RepickReasonPostUseSkill  RepickReason = "post_use_skill"
	RepickReasonDamaged       RepickReason = "damaged"
	RepickReasonStatusApplied RepickReason = "status_applied"
	RepickReasonStatusExpired RepickReason = "status_expired"
	RepickReasonControlChange RepickReason = "control_change"
	RepickReasonSweep         RepickReason = "sweep"
)

// randSource lets tests inject a deterministic RNG. In production the
// picker uses package-level math/rand.
type randSource interface {
	Intn(n int) int
}

// cooldownReader is the picker's read-only view onto cooldown state. The
// production cooldownRegistry satisfies this; tests can substitute fakes.
type cooldownReader interface {
	IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) bool
	Remaining(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) time.Duration
}

// mobSkillFetcher abstracts the atlas-data REST lookup so tests can run
// without HTTP. Production passes mobskill.GetByIdAndLevel-derived closure.
type mobSkillFetcher func(skillId, skillLevel uint16) (mobskill.Model, error)

// monsterInfoFetcher abstracts information.GetById lookup for the picker.
type monsterInfoFetcher func(monsterId uint32) (information.Model, error)

// pickerRelevantStatuses are the monster status-name strings whose apply or
// expire flips picker eligibility. SEAL gates everything; the *_REFLECT and
// *_IMMUNITY statuses gate stacking checks for those skill categories.
var pickerRelevantStatuses = map[string]struct{}{
	string(monster2.TemporaryStatTypeSeal):               {},
	string(monster2.TemporaryStatTypeSealSkill):          {},
	string(monster2.TemporaryStatTypeWeaponAttackImmune): {},
	string(monster2.TemporaryStatTypeMagicAttackImmune):  {},
	string(monster2.TemporaryStatTypeWeaponCounter):      {},
	string(monster2.TemporaryStatTypeMagicCounter):       {},
}

// isPickerRelevantStatus reports whether the given status-name string
// belongs to the picker-relevant set.
func isPickerRelevantStatus(name string) bool {
	_, ok := pickerRelevantStatuses[name]
	return ok
}

// effectTouchesPicker returns true if any status name inside the effect's
// status map is picker-relevant.
func effectTouchesPicker(e StatusEffect) bool {
	for k := range e.Statuses() {
		if isPickerRelevantStatus(k) {
			return true
		}
	}
	return false
}

// pickNextSkill is the pure picker. It iterates the monster's skill list,
// runs the eligibility gates from PRD §FR-2, and rolls each candidate's prop
// independently. First successful roll wins. No mutations.
//
// The picker computes nextEligibleRepickAtMs as the minimum cooldown expiry
// across skills gated only by cooldown. If none, returns 0 (sentinel: sweep
// skips this monster).
func pickNextSkill(
	l logrus.FieldLogger,
	ctx context.Context,
	t tenant.Model,
	m Model,
	info monsterInfoFetcher,
	skills mobSkillFetcher,
	cooldown cooldownReader,
	rng randSource,
	nowMs int64,
) Decision {
	ma, err := info(m.MonsterId())
	if err != nil {
		l.WithError(err).Debugf("Picker: cannot fetch info for monster [%d]; treating as no-skill.", m.UniqueId())
		return Decision{}
	}
	if len(ma.Skills()) == 0 {
		return Decision{}
	}

	// Sealed monsters cannot fire any skill; emit sentinel.
	if m.HasStatusEffect("SEAL") {
		l.Debugf("Picker: monster [%d] is SEALed; no candidates.", m.UniqueId())
		return Decision{}
	}

	chosen := Decision{}
	var nextRepick int64

	for _, s := range ma.Skills() {
		// Defensive byte-overflow guard. atlas-data Skill carries uint32; we
		// must narrow to byte for the wire/packet. Anything beyond 255 is
		// malformed data — log and skip.
		if s.Id > 255 || s.Level > 255 {
			l.Warnf("Picker: monster [%d] skill (%d, %d) out of byte range; skipping.", m.UniqueId(), s.Id, s.Level)
			continue
		}
		skillId16 := uint16(s.Id)
		skillLevel16 := uint16(s.Level)

		// AREA_POISON exclusion. TODO(spec-task-3): remove when the mist
		// executor lands so the picker can fire mist skills.
		if skillId16 == monster2.SkillTypeAreaPoison {
			l.Debugf("Picker: monster [%d] skipping AREA_POISON (skill type %d) until spec-task-3.", m.UniqueId(), skillId16)
			continue
		}

		sd, err := skills(skillId16, skillLevel16)
		if err != nil {
			l.WithError(err).Debugf("Picker: monster [%d] cannot fetch skill (%d,%d); skipping.", m.UniqueId(), skillId16, skillLevel16)
			continue
		}

		// Cooldown gate.
		if cooldown.IsOnCooldown(ctx, t, m.UniqueId(), byte(skillId16)) {
			rem := cooldown.Remaining(ctx, t, m.UniqueId(), byte(skillId16))
			if rem > 0 {
				expiry := nowMs + rem.Milliseconds()
				if nextRepick == 0 || expiry < nextRepick {
					nextRepick = expiry
				}
			}
			l.Debugf("Picker: monster [%d] skill [%d] on cooldown (rem=%s); skipping.", m.UniqueId(), skillId16, rem)
			continue
		}

		// HP threshold gate. sd.Hp() is the maximum HP% at which the skill
		// becomes eligible (mirrors processor.go:486). Zero = no gate.
		if sd.Hp() > 0 && m.HpPercentage() > sd.Hp() {
			l.Debugf("Picker: monster [%d] HP %d%% > skill [%d] threshold %d%%; skipping.", m.UniqueId(), m.HpPercentage(), skillId16, sd.Hp())
			continue
		}

		// MP gate.
		if sd.MpCon() > 0 && m.Mp() < sd.MpCon() {
			l.Debugf("Picker: monster [%d] insufficient MP (%d < %d) for skill [%d]; skipping.", m.UniqueId(), m.Mp(), sd.MpCon(), skillId16)
			continue
		}

		// Reflect/immunity already-active gate (mirrors processor.go:519-527).
		category := monster2.SkillCategory(skillId16)
		if category == monster2.SkillCategoryImmunity || category == monster2.SkillCategoryReflect {
			statusName := monster2.SkillTypeToStatusName(skillId16)
			if statusName != "" && m.HasStatusEffect(string(statusName)) {
				l.Debugf("Picker: monster [%d] already has %s; skipping skill [%d].", m.UniqueId(), statusName, skillId16)
				continue
			}
		}

		// Prop roll. Per PRD §FR-3, first success wins.
		prop := int(sd.Prop())
		if prop <= 0 {
			continue
		}
		if prop > 100 {
			prop = 100
		}
		if rng.Intn(100) < prop {
			chosen = Decision{
				SkillId:    byte(skillId16),
				SkillLevel: byte(skillLevel16),
			}
			break
		}
	}

	chosen.DecidedAtMs = nowMs
	chosen.NextEligibleRepickAtMs = nextRepick
	return chosen
}

// repickAndEmit reads the monster from the registry, runs the picker, writes
// the decision back into the registry, and emits a NEXT_SKILL_DECIDED event.
// Always emits — even if the new decision is the sentinel or unchanged —
// because atlas-channel's inbox is single-use and stale-cache-coherent: a
// missed emission would leave a stale prediction in place. Logs at debug
// per-run; logs at info level on sentinel↔non-sentinel transitions.
func (p *ProcessorImpl) repickAndEmit(uniqueId uint32, reason RepickReason) error {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil {
		// Monster gone (destroyed between trigger and call). Drop quietly.
		return nil
	}

	infoFn := func(monsterId uint32) (information.Model, error) {
		return information.GetById(p.l)(p.ctx)(monsterId)
	}
	skillsFn := func(skillId, skillLevel uint16) (mobskill.Model, error) {
		return mobskill.GetByIdAndLevel(p.l)(p.ctx)(skillId, skillLevel)
	}
	rng := pickerRNG{}

	prev := m.NextSkillDecision()
	now := time.Now().UnixMilli()
	d := pickNextSkill(p.l, p.ctx, p.t, m, infoFn, skillsFn, GetCooldownRegistry(), rng, now)

	// Sentinel↔non-sentinel transition logging at info level.
	wasSentinel := prev.skillId == 0
	isSentinel := d.IsSentinel()
	if wasSentinel != isSentinel {
		if isSentinel {
			p.l.Infof("Picker: monster [%d] transition non-sentinel(%d)→sentinel reason=%s.", m.UniqueId(), prev.skillId, reason)
		} else {
			p.l.Infof("Picker: monster [%d] transition sentinel→casting(%d) reason=%s.", m.UniqueId(), d.SkillId, reason)
		}
	}

	nd := nextSkillDecision{
		skillId:                d.SkillId,
		skillLevel:             d.SkillLevel,
		decidedAtMs:            d.DecidedAtMs,
		nextEligibleRepickAtMs: d.NextEligibleRepickAtMs,
	}
	updated, err := GetMonsterRegistry().SetNextSkillDecision(p.t, uniqueId, nd)
	if err != nil {
		p.l.WithError(err).Errorf("Picker: failed to store decision for monster [%d].", uniqueId)
		// Continue and emit anyway: the consumer is the source of truth for
		// atlas-channel's inbox, and a stale local store will repair on the
		// next picker run.
	}
	_ = updated // decision is in-memory only; not persisted (see SetNextSkillDecision doc)

	// Always emit, even on sentinel/unchanged decisions, to keep atlas-channel
	// inbox coherent.
	if err := p.emit(EnvEventTopicMonsterStatus, nextSkillDecidedStatusEventProvider(m, nd)); err != nil {
		p.l.WithError(err).Errorf("Picker: failed to emit NEXT_SKILL_DECIDED for monster [%d].", uniqueId)
		return err
	}
	return nil
}

// pickerRNG is the production RNG. Wraps math/rand for the randSource
// interface. Tests inject fakeRand instead.
type pickerRNG struct{}

func (pickerRNG) Intn(n int) int { return rand.Intn(n) }
