package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MonsterSkillPickerSweepInterval is the cadence at which
// MonsterSkillPickerSweepTask runs.
const MonsterSkillPickerSweepInterval = 1500 * time.Millisecond

// MonsterSkillPickerSweepTask periodically scans all live monsters and
// re-runs the skill picker for any monster whose next-eligible-repick
// timestamp has elapsed.
type MonsterSkillPickerSweepTask struct {
	l           logrus.FieldLogger
	ctx         context.Context
	interval    time.Duration
	nowFn       func() int64
	repickFn    func(t tenant.Model, uniqueId uint32) error
	hasSkillsFn func(t tenant.Model, monsterId uint32) bool
}

// NewMonsterSkillPickerSweepTask constructs a sweep task with production
// implementations of repickFn and hasSkillsFn.
func NewMonsterSkillPickerSweepTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *MonsterSkillPickerSweepTask {
	l.Infof("Initializing monster skill picker sweep task to run every %dms.", interval.Milliseconds())
	tk := &MonsterSkillPickerSweepTask{
		l:        l,
		ctx:      ctx,
		interval: interval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
	}
	tk.repickFn = func(t tenant.Model, uniqueId uint32) error {
		tctx := tenant.WithContext(tk.ctx, t)
		return NewProcessor(tk.l, tctx).RepickAndEmit(uniqueId, RepickReasonSweep)
	}
	tk.hasSkillsFn = func(t tenant.Model, monsterId uint32) bool {
		tctx := tenant.WithContext(tk.ctx, t)
		ma, err := information.GetById(tk.l)(tctx)(monsterId)
		if err != nil {
			return false
		}
		return len(ma.Skills()) > 0
	}
	return tk
}

// SleepTime returns the task's run interval.
func (tk *MonsterSkillPickerSweepTask) SleepTime() time.Duration { return tk.interval }

// Run scans all live monsters and repicks any whose next-eligible-repick
// timestamp has elapsed and whose template has at least one skill.
func (tk *MonsterSkillPickerSweepTask) Run() {
	monsters := GetMonsterRegistry().GetMonsters()
	now := tk.nowFn()
	skillCache := make(map[uuid.UUID]map[uint32]bool)

	for ten, mons := range monsters {
		for _, m := range mons {
			d := m.NextSkillDecision()
			if d.nextEligibleRepickAtMs == 0 || d.nextEligibleRepickAtMs > now {
				continue
			}
			if !m.ControllerHasAggro() {
				continue
			}
			templateId := m.MonsterId()
			tenantId := ten.Id()
			if skillCache[tenantId] == nil {
				skillCache[tenantId] = make(map[uint32]bool)
			}
			has, cached := skillCache[tenantId][templateId]
			if !cached {
				has = tk.hasSkillsFn(ten, templateId)
				skillCache[tenantId][templateId] = has
			}
			if !has {
				continue
			}
			if err := tk.repickFn(ten, m.UniqueId()); err != nil {
				tk.l.WithError(err).Errorf("Sweep picker: monster [%d] re-pick failed.", m.UniqueId())
			}
		}
	}
}
