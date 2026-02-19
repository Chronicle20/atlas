package monster

import (
	"atlas-monsters/kafka/producer"
	"atlas-monsters/monster/drop"
	"context"
	"math/rand"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type DropTimerTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewDropTimerTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *DropTimerTask {
	l.Infof("Initializing drop timer task to run every %dms.", interval.Milliseconds())
	return &DropTimerTask{l: l, ctx: ctx, interval: interval}
}

func (t *DropTimerTask) Run() {
	now := time.Now()
	entries := GetDropTimerRegistry().GetAll()
	for key, entry := range entries {
		t.processEntry(now, key.Tenant, key.MonsterId, entry)
	}
}

func (t *DropTimerTask) processEntry(now time.Time, ten tenant.Model, uniqueId uint32, e DropTimerEntry) {
	// Determine next eligible drop time
	var nextEligible time.Time
	if !e.LastHitAt().IsZero() && e.LastHitAt().After(e.LastDropAt()) {
		// Monster was hit since last drop - next eligible is lastHitAt + dropPeriod
		nextEligible = e.LastHitAt().Add(e.DropPeriod())
	} else {
		// No recent hit - next eligible is lastDropAt + dropPeriod
		nextEligible = e.LastDropAt().Add(e.DropPeriod())
	}

	if now.Before(nextEligible) {
		return
	}

	// Verify monster is still alive
	m, err := GetMonsterRegistry().GetMonster(ten, uniqueId)
	if err != nil || !m.Alive() {
		GetDropTimerRegistry().Unregister(ten, uniqueId)
		return
	}

	tctx := tenant.WithContext(t.ctx, ten)
	t.produceDrop(tctx, m, e)
	GetDropTimerRegistry().UpdateLastDrop(ten, uniqueId, now)
}

func (t *DropTimerTask) produceDrop(ctx context.Context, m Model, e DropTimerEntry) {
	ds, err := drop.GetByMonsterId(t.l)(ctx)(e.MonsterId())
	if err != nil {
		t.l.WithError(err).Errorf("Unable to fetch drop table for friendly monster [%d] (template [%d]).", m.UniqueId(), e.MonsterId())
		return
	}

	// Filter out quest-specific drops (no character context for quest checks)
	filtered := make([]drop.Model, 0, len(ds))
	for _, d := range ds {
		if d.QuestId() == 0 {
			filtered = append(filtered, d)
		}
	}

	f := m.Field()
	var dropCount uint32
	for _, d := range filtered {
		if rand.Int31n(999999) >= int32(d.Chance()) {
			continue
		}

		quantity := uint32(1)
		if d.MaximumQuantity() != 1 && d.MaximumQuantity() > d.MinimumQuantity() {
			quantity = uint32(rand.Int31n(int32(d.MaximumQuantity()-d.MinimumQuantity())+1)) + d.MinimumQuantity()
		}

		var itemId uint32
		var mesos uint32
		if d.ItemId() == 0 {
			mesos = quantity
			quantity = 0
		} else {
			itemId = d.ItemId()
		}

		cp := drop.SpawnDropCommandProvider(f, itemId, quantity, mesos, m.X(), m.Y(), m.UniqueId())
		err := producer.ProviderImpl(t.l)(ctx)(drop.EnvCommandTopicDrop)(cp)
		if err != nil {
			t.l.WithError(err).Errorf("Unable to emit drop for friendly monster [%d].", m.UniqueId())
		} else {
			dropCount++
		}
	}

	if dropCount > 0 {
		err := producer.ProviderImpl(t.l)(ctx)(EnvEventTopicMonsterStatus)(friendlyDropStatusEventProvider(f, m.UniqueId(), e.MonsterId(), dropCount))
		if err != nil {
			t.l.WithError(err).Errorf("Unable to emit friendly drop event for monster [%d].", m.UniqueId())
		}
	}
}

func (t *DropTimerTask) SleepTime() time.Duration {
	return t.interval
}
