package monster

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"atlas-monsters/monster/information"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// MonsterRecoveryInterval is the cadence at which MonsterRecoveryTask runs.
// 10s mirrors v83 reference behavior; not configurable per tenant (PRD §2 non-goal).
const MonsterRecoveryInterval = 10 * time.Second

// recoveryApplyFn is the registry-side recovery write. Production wires
// (*Registry).ApplyRecovery; tests inject fakes.
type recoveryApplyFn func(t tenant.Model, uniqueId uint32, hpRecovery, mpRecovery uint32, nowMs int64) (Model, bool, bool, error)

// recoveryEmitFn publishes the HP-bar refresh event (DamageSourceHeal, damage=0).
// Production wraps producer.ProviderImpl(...); tests intercept.
type recoveryEmitFn func(t tenant.Model, m Model) error

// recoveryMpEmitFn publishes the MP_CHANGED event for applied MP regen.
// Production wraps producer.ProviderImpl(...); tests intercept.
type recoveryMpEmitFn func(t tenant.Model, m Model, amount uint32) error

// recoveryInfoFn fetches the monster's template information.Model. Production
// wraps information.GetById; tests inject fakes.
type recoveryInfoFn func(t tenant.Model, monsterId uint32) (information.Model, error)

// MonsterRecoveryTask periodically applies HP/MP recovery to all live monsters
// across all tenants. HP recovery is gated by the 10s damage-idle window;
// MP recovery is unconditional. Recovery values come from atlas-data WZ
// (info/hpRecovery, info/mpRecovery), exposed via information.Model.
type MonsterRecoveryTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
	nowFn    func() int64
	infoFn   recoveryInfoFn
	applyFn  recoveryApplyFn
	emitFn   recoveryEmitFn
	mpEmitFn recoveryMpEmitFn
}

// NewMonsterRecoveryTask wires production implementations.
func NewMonsterRecoveryTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *MonsterRecoveryTask {
	l.Infof("Initializing monster recovery task to run every %dms.", interval.Milliseconds())
	tk := &MonsterRecoveryTask{
		l:        l,
		ctx:      ctx,
		interval: interval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
	}
	tk.infoFn = func(t tenant.Model, monsterId uint32) (information.Model, error) {
		tctx := tenant.WithContext(tk.ctx, t)
		return information.NewProcessor(tk.l, tctx).GetById(monsterId)
	}
	tk.applyFn = GetMonsterRegistry().ApplyRecovery
	tk.emitFn = func(t tenant.Model, m Model) error {
		tctx := tenant.WithContext(tk.ctx, t)
		return producer.ProviderImpl(tk.l)(tctx)(EnvEventTopicMonsterStatus)(
			damagedStatusEventProvider(m, m.UniqueId(), m.UniqueId(), false, DamageSourceHeal, m.DamageSummary()),
		)
	}
	tk.mpEmitFn = func(t tenant.Model, m Model, amount uint32) error {
		tctx := tenant.WithContext(tk.ctx, t)
		return producer.ProviderImpl(tk.l)(tctx)(EnvEventTopicMonsterStatus)(
			mpChangedStatusEventProvider(m, 0, 0, MpChangeReasonRecovery, amount),
		)
	}
	// Compile-time guard so unused imports fail loudly if any wiring drifts.
	var _ model.Provider[[]kafka.Message] = damagedStatusEventProvider(Model{}, 0, 0, false, "", nil)
	return tk
}

// SleepTime returns the task's run interval.
func (tk *MonsterRecoveryTask) SleepTime() time.Duration { return tk.interval }

// Run iterates every live monster across every tenant and applies recovery
// per the rules in PRD §FR-5. Errors per-monster are logged at Debug and skip
// only that monster — never crash the tick.
func (tk *MonsterRecoveryTask) Run() {
	monsters := GetMonsterRegistry().GetMonsters()
	nowMs := tk.nowFn()
	infoCache := make(map[uuid.UUID]map[uint32]information.Model)

	for ten, mons := range monsters {
		tenantId := ten.Id()
		if infoCache[tenantId] == nil {
			infoCache[tenantId] = make(map[uint32]information.Model)
		}
		for _, m := range mons {
			if !m.Alive() {
				continue
			}
			if m.Hp() == m.MaxHp() && m.Mp() == m.MaxMp() {
				continue
			}

			info, ok := infoCache[tenantId][m.MonsterId()]
			if !ok {
				fetched, err := tk.infoFn(ten, m.MonsterId())
				if err != nil {
					tk.l.WithError(err).Debugf(
						"Recovery: cannot fetch info for monster [%d]; skipping.", m.UniqueId())
					continue
				}
				info = fetched
				infoCache[tenantId][m.MonsterId()] = info
			}

			hpR := info.HpRecovery()
			mpR := info.MpRecovery()
			if hpR == 0 && mpR == 0 {
				continue
			}

			updated, hpApplied, mpApplied, err := tk.applyFn(ten, m.UniqueId(), hpR, mpR, nowMs)
			if err != nil {
				tk.l.WithError(err).Debugf(
					"Recovery: apply failed for monster [%d]; skipping.", m.UniqueId())
				continue
			}
			if hpApplied {
				if err := tk.emitFn(ten, updated); err != nil {
					tk.l.WithError(err).Debugf(
						"Recovery: HP-bar emit failed for monster [%d].", updated.UniqueId())
				}
			}
			if mpApplied {
				// Best-effort applied amount from the pre/post snapshots;
				// the mirror consumer only reads MonsterMpAfter, which the
				// post model carries authoritatively.
				var amount uint32
				if updated.Mp() > m.Mp() {
					amount = updated.Mp() - m.Mp()
				}
				if err := tk.mpEmitFn(ten, updated, amount); err != nil {
					tk.l.WithError(err).Debugf(
						"Recovery: MP_CHANGED emit failed for monster [%d].", updated.UniqueId())
				}
			}
		}
	}
}
