package monster

import (
	"atlas-monsters/kafka/producer"
	"atlas-monsters/monster/information"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// taskEmitter publishes a kafka message provider on behalf of a tenant.
// Injected for tests.
type taskEmitter func(t tenant.Model, topic string, provider model.Provider[[]kafka.Message]) error

// bossLookupFn fetches whether a template is a boss for a given tenant.
// The tenant is required because atlas-data is tenant-scoped — the upstream
// REST middleware rejects requests without a TENANT_ID header.
type bossLookupFn func(t tenant.Model, monsterTemplateId uint32) bool

type MonsterAggroDecayTask struct {
	l            logrus.FieldLogger
	ctx          context.Context
	interval     time.Duration
	bossLookupFn bossLookupFn
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
	tk.bossLookupFn = func(t tenant.Model, monsterTemplateId uint32) bool {
		tctx := tenant.WithContext(tk.ctx, t)
		ma, err := information.GetById(tk.l)(tctx)(monsterTemplateId)
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
	bossCache := make(map[uuid.UUID]map[uint32]bool)
	nowMs := tk.nowFn()

	for ten, mons := range monsters {
		tenantId := ten.Id()
		if bossCache[tenantId] == nil {
			bossCache[tenantId] = make(map[uint32]bool)
		}
		for _, m := range mons {
			templateId := m.MonsterId()
			isBoss, ok := bossCache[tenantId][templateId]
			if !ok {
				isBoss = tk.bossLookupFn(ten, templateId)
				bossCache[tenantId][templateId] = isBoss
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
