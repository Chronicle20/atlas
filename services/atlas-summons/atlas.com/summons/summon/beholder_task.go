package summon

import (
	buffmsg "atlas-summons/buff"
	charmsg "atlas-summons/character"
	producer "atlas-summons/kafka/producer"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// BeholderTask periodically heals the owner and re-applies the Beholder buff for
// every deployed Beholder (BUFF_AURA) summon whose heal/buff timer is due. It runs
// only on the leader-elected pod (registered from main.go's registerSweepTasks),
// so each due tick fires exactly once.
type BeholderTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
	// emit publishes a kafka message provider to a topic. It is a field so tests
	// can substitute a capturing emitter and avoid a real kafka publish; production
	// uses producer.ProviderImpl.
	emit func(topic string, provider model.Provider[[]kafka.Message]) error
}

func NewBeholderTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *BeholderTask {
	return &BeholderTask{
		l:        l,
		ctx:      ctx,
		interval: interval,
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			return producer.ProviderImpl(l)(ctx)(topic)(provider)
		},
	}
}

func (t *BeholderTask) SleepTime() time.Duration { return t.interval }

// Run enumerates every stored summon grouped by tenant and, for each deployed
// Beholder, emits a CHANGE_HP (heal) and/or buff APPLY when its respective timer
// is due, then advances and persists the timer. Reading from the registry each
// tick means a despawned Beholder (removed from the registry) is never swept
// again. Zero-valued intervals are skipped to avoid a spin (advancing by 0 would
// keep the timer perpetually due).
func (t *BeholderTask) Run() {
	all, err := GetRegistry().GetAll(t.ctx)
	if err != nil {
		t.l.WithError(err).Errorf("Beholder sweep unable to enumerate summons.")
		return
	}
	now := time.Now()
	for ten, ms := range all {
		tctx := tenant.WithContext(t.ctx, ten)
		for _, m := range ms {
			if !m.IsBeholder() {
				continue
			}
			t.sweepHeal(tctx, ten, m, now)
			t.sweepBuff(tctx, ten, m, now)
		}
	}
}

func (t *BeholderTask) sweepHeal(ctx context.Context, ten tenant.Model, m Model, now time.Time) {
	interval := m.HealInterval()
	if interval <= 0 {
		return
	}
	if m.NextHealAt().IsZero() || now.Before(m.NextHealAt()) {
		return
	}
	f := m.Field()
	if err := t.emit(charmsg.EnvCommandTopic, charmsg.ChangeHPProvider(f.WorldId(), f.ChannelId(), m.OwnerCharacterId(), m.HealAmount())); err != nil {
		t.l.WithError(err).Warnf("Beholder sweep failed to heal owner [%d] of summon [%d].", m.OwnerCharacterId(), m.Id())
		return
	}
	next := m.NextHealAt().Add(interval)
	if _, err := GetRegistry().Update(ctx, ten, m.Id(), func(cur Model) Model {
		return Clone(cur).SetNextHealAt(next).Build()
	}); err != nil {
		t.l.WithError(err).Warnf("Beholder sweep failed to persist NextHealAt for summon [%d].", m.Id())
	}
}

func (t *BeholderTask) sweepBuff(ctx context.Context, ten tenant.Model, m Model, now time.Time) {
	interval := m.BuffInterval()
	if interval <= 0 {
		return
	}
	if m.NextBuffAt().IsZero() || now.Before(m.NextBuffAt()) {
		return
	}
	changes := make([]buffmsg.StatChange, 0, len(m.BuffChanges()))
	for _, c := range m.BuffChanges() {
		changes = append(changes, buffmsg.StatChange{Type: c.Type, Amount: c.Amount})
	}
	if err := t.emit(buffmsg.EnvCommandTopic, buffmsg.ApplyProvider(m.Field(), m.OwnerCharacterId(), m.OwnerCharacterId(), m.BuffSourceId(), m.BuffLevel(), m.BuffDuration(), changes)); err != nil {
		t.l.WithError(err).Warnf("Beholder sweep failed to apply buff to owner [%d] of summon [%d].", m.OwnerCharacterId(), m.Id())
		return
	}
	// Emit the SKILL status event so the channel rebroadcasts the SummonSkill
	// buff-pulse visual map-wide. Cosmic (Character.java:4487) plays the HEX buff
	// pulse at stance (random*3)+6, i.e. 6-8; the sweep fires once per due tick on
	// the leader pod and can't replicate per-tick client-side randomization, so it
	// uses the lowest valid buff-pulse stance (6) deterministically. A failure here
	// is non-fatal: the buff already applied, so the timer must still advance.
	const beholderBuffStance = byte(6)
	if err := t.emit(EnvEventTopicSummonStatus, skillEventProvider(m, beholderBuffStance)); err != nil {
		t.l.WithError(err).Warnf("Beholder sweep failed to emit skill effect for summon [%d].", m.Id())
	}
	next := m.NextBuffAt().Add(interval)
	if _, err := GetRegistry().Update(ctx, ten, m.Id(), func(cur Model) Model {
		return Clone(cur).SetNextBuffAt(next).Build()
	}); err != nil {
		t.l.WithError(err).Warnf("Beholder sweep failed to persist NextBuffAt for summon [%d].", m.Id())
	}
}
