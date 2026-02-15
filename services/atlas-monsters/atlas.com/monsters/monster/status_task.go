package monster

import (
	"atlas-monsters/kafka/producer"
	"context"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
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

	if se.HasStatus("POISON") {
		totalDamage += t.calculatePoisonDamage(m, se)
	}
	if se.HasStatus("VENOM") {
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
	ds, err := GetMonsterRegistry().ApplyDamage(ten, se.SourceCharacterId(), totalDamage, m.UniqueId())
	if err != nil {
		t.l.WithError(err).Errorf("Unable to apply DoT damage to monster [%d].", m.UniqueId())
		return
	}

	// Update last tick
	_, _ = GetMonsterRegistry().UpdateStatusEffectLastTick(ten, m.UniqueId(), se.EffectId(), time.Now())

	// Emit damaged event
	_ = producer.ProviderImpl(t.l)(ctx)(EnvEventTopicMonsterStatus)(damagedStatusEventProvider(ds.Monster, se.SourceCharacterId(), false, ds.Monster.DamageSummary()))
}

func (t *StatusExpirationTask) calculatePoisonDamage(m Model, se StatusEffect) uint32 {
	// Poison damage formula: maxHP / (70 - skillLevel)
	divisor := int32(70) - int32(se.SourceSkillLevel())
	if divisor <= 0 {
		divisor = 1
	}
	return m.MaxHp() / uint32(divisor)
}

func (t *StatusExpirationTask) calculateVenomDamage(se StatusEffect) uint32 {
	// Venom damage is the stat value applied to the effect
	if val, ok := se.Statuses()["VENOM"]; ok && val > 0 {
		return uint32(val)
	}
	return 0
}

func (t *StatusExpirationTask) SleepTime() time.Duration {
	return t.interval
}
