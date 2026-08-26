package monster

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type StatusExpirationTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewStatusExpirationTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *StatusExpirationTask {
	l.Infof("Initializing status expiration task to run every %dms.", interval.Milliseconds())
	return &StatusExpirationTask{l: l, ctx: ctx, interval: interval}
}

func (t *StatusExpirationTask) Run() {
	monsters := GetMonsterRegistry().GetMonsters()
	for ten, mons := range monsters {
		for _, m := range mons {
			if len(m.StatusEffects()) == 0 {
				continue
			}
			t.processMonsterEffects(ten, m)
		}
	}
}

func (t *StatusExpirationTask) processMonsterEffects(ten tenant.Model, m Model) {
	tctx := tenant.WithContext(t.ctx, ten)

	for _, se := range m.StatusEffects() {
		if se.Expired() {
			updated, err := GetMonsterRegistry().CancelStatusEffect(ten, m.UniqueId(), se.EffectId())
			if err != nil {
				t.l.WithError(err).Errorf("Unable to expire status effect [%s] from monster [%d].", se.EffectId(), m.UniqueId())
				continue
			}
			_ = producer.ProviderImpl(t.l)(tctx)(EnvEventTopicMonsterStatus)(statusEffectExpiredEventProvider(updated, se))
			if effectTouchesPicker(se) {
				if err := NewProcessor(t.l, tctx).RepickAndEmit(updated.UniqueId(), RepickReasonStatusExpired); err != nil {
					t.l.WithError(err).Warnf("Status-expired picker: monster [%d] re-pick failed.", updated.UniqueId())
				}
			}
			continue
		}

		// Process DoT ticks
		if se.ShouldTick() {
			t.processDoTTick(ten, tctx, m, se)
		}
	}
}

func (t *StatusExpirationTask) processDoTTick(ten tenant.Model, ctx context.Context, m Model, se StatusEffect) {
	var totalDamage uint32

	if se.HasStatus(StatusPoison) {
		totalDamage += t.calculatePoisonDamage(m, se)
	}
	if se.HasStatus(StatusVenom) {
		totalDamage += t.calculateVenomDamage(se)
	}

	if totalDamage == 0 {
		// Update last tick even if no damage (for non-damage ticking effects)
		_, _ = GetMonsterRegistry().UpdateStatusEffectLastTick(ten, m.UniqueId(), se.EffectId(), time.Now())
		return
	}

	// Kill prevention: cap damage at currentHP - 1
	current, err := GetMonsterRegistry().GetMonster(ten, m.UniqueId())
	if err != nil || !current.Alive() {
		return
	}
	if totalDamage >= current.Hp() {
		totalDamage = current.Hp() - 1
	}
	if totalDamage == 0 {
		_, _ = GetMonsterRegistry().UpdateStatusEffectLastTick(ten, m.UniqueId(), se.EffectId(), time.Now())
		return
	}

	// Apply damage
	ds, err := GetMonsterRegistry().ApplyDamage(ten, se.SourceCharacterId(), totalDamage, m.UniqueId(), time.Now().UnixMilli())
	if err != nil {
		t.l.WithError(err).Errorf("Unable to apply DoT damage to monster [%d].", m.UniqueId())
		return
	}

	// Update last tick
	_, _ = GetMonsterRegistry().UpdateStatusEffectLastTick(ten, m.UniqueId(), se.EffectId(), time.Now())

	// Emit damaged event
	_ = producer.ProviderImpl(t.l)(ctx)(EnvEventTopicMonsterStatus)(damagedStatusEventProvider(ds.Monster, se.SourceCharacterId(), se.SourceCharacterId(), false, DamageSourceDamageOverTime, totalDamage, ds.Monster.DamageSummary()))

	// FR-2.4: DoT never enters damageCore (it caps at currentHp-1 and cannot
	// kill), but the self-destruct threshold is above zero, so a poison tick
	// can still cross it. The kill-prevention cap above is untouched — poison
	// still cannot reduce a mob to 0 HP; it can only trip a detonation
	// (task-253 design §2.5).
	if ma, ierr := resolveMonsterInformation(t.l, ctx, m.MonsterId()); ierr == nil {
		sd := ma.SelfDestruction()
		if sd.OnHpThreshold() && int64(ds.Monster.Hp()) <= int64(sd.Hp()) {
			NewProcessor(t.l, ctx).SelfDestruct(m.UniqueId(), se.SourceCharacterId(), TriggerThreshold)
		}
	}
}

// calculatePoisonDamage reads the per-tick magnitude ApplyStatusEffect
// resolved and stored on the effect (ResolvePoisonDamage). Reading the stored
// value rather than recomputing is what keeps the damage applied here and the
// magnitude the client renders from identical.
//
// The recompute is a fallback for an effect that reached the registry without
// passing through ApplyStatusEffect (older persisted state across a rolling
// deploy); it is the same function, so it cannot diverge.
func (t *StatusExpirationTask) calculatePoisonDamage(m Model, se StatusEffect) uint32 {
	if val, ok := se.Statuses()[StatusPoison]; ok && val > 0 {
		return uint32(val)
	}
	return uint32(ResolvePoisonDamage(m.MaxHp(), se.SourceSkillLevel()))
}

func (t *StatusExpirationTask) calculateVenomDamage(se StatusEffect) uint32 {
	// Venom damage is the stat value applied to the effect
	if val, ok := se.Statuses()[StatusVenom]; ok && val > 0 {
		return uint32(val)
	}
	return 0
}

func (t *StatusExpirationTask) SleepTime() time.Duration {
	return t.interval
}
