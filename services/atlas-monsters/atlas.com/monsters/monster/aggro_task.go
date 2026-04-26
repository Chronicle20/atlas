package monster

import (
	"atlas-monsters/kafka/producer"
	"atlas-monsters/monster/information"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// taskEmitter publishes a kafka message provider on behalf of a tenant.
// Injected for tests.
type taskEmitter func(t tenant.Model, topic string, provider model.Provider[[]kafka.Message]) error

type MonsterAggroDecayTask struct {
	l            logrus.FieldLogger
	ctx          context.Context
	interval     time.Duration
	bossLookupFn func(monsterTemplateId uint32) bool
	emit         taskEmitter
	nowFn        func() int64
}

func NewMonsterAggroDecayTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *MonsterAggroDecayTask {
	l.Infof("Initializing monster aggro decay task to run every %dms.", interval.Milliseconds())
	tk := &MonsterAggroDecayTask{
		l:        l,
		ctx:      ctx,
		interval: interval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
	}
	tk.bossLookupFn = func(monsterTemplateId uint32) bool {
		ma, err := information.GetById(tk.l)(tk.ctx)(monsterTemplateId)
		if err != nil {
			// Best-effort: treat as non-boss so decay proceeds.
			return false
		}
		return ma.Boss()
	}
	tk.emit = func(t tenant.Model, topic string, provider model.Provider[[]kafka.Message]) error {
		tctx := tenant.WithContext(tk.ctx, t)
		return producer.ProviderImpl(tk.l)(tctx)(topic)(provider)
	}
	return tk
}

func (tk *MonsterAggroDecayTask) SleepTime() time.Duration {
	return tk.interval
}

func (tk *MonsterAggroDecayTask) Run() {
	monsters := GetMonsterRegistry().GetMonsters()
	bossCache := make(map[uint32]bool)
	nowMs := tk.nowFn()

	for ten, mons := range monsters {
		for _, m := range mons {
			templateId := m.MonsterId()
			isBoss, ok := bossCache[templateId]
			if !ok {
				isBoss = tk.bossLookupFn(templateId)
				bossCache[templateId] = isBoss
			}
			if isBoss {
				continue
			}
			entries := m.DamageEntries()
			if len(entries) == 0 {
				continue
			}
			needsWork := false
			for _, e := range entries {
				if IsAggroIdle(e, nowMs) {
					needsWork = true
					break
				}
			}
			if !needsWork {
				continue
			}
			summary, err := GetMonsterRegistry().DecayDamageEntries(ten, m.UniqueId(), nowMs)
			if err != nil {
				tk.l.WithError(err).Errorf("Decay failed for monster [%d].", m.UniqueId())
				continue
			}
			if summary.AggroFlippedOff {
				_ = tk.emit(ten, EnvEventTopicMonsterStatus, aggroChangedStatusEventProvider(summary.Monster, summary.ControllerCharacterId, false))
				tk.l.Debugf("Aggro decay flipped controller [%d] to passive for monster [%d].", summary.ControllerCharacterId, summary.Monster.UniqueId())
			}
		}
	}
}
