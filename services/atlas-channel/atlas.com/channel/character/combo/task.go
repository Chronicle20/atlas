package combo

import (
	"atlas-channel/character/buff"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// DecayTick expires idle Aran combos.
//
// It walks the combo mirror only -- never sessions, never a tenant list -- so
// a channel with no Aran in combat does no work per tick (task-217 FR-4.5).
// It deliberately sends NO packet for an expiry: DrawCombo early-returns on a
// non-positive count without releasing its digit layers, so SHOW_COMBO 0
// cannot clear the HUD. The client runs the same idle timer and clears itself
// (design.md §2.5, §5.3); the server's job is to agree with it.
type DecayTick struct {
	l          logrus.FieldLogger
	ctx        context.Context
	interval   time.Duration
	envContext func(context.Context) context.Context
}

// NewDecayTick builds the idle-combo sweep. envContext originates the
// environment that owns each expired combo's tenant (falling back to this
// pod's own, env.Self(), when the tenant is unknown) onto that combo's
// per-character context before the buff-cancel Kafka event is produced --
// character/combo is outside env-domain-guard's permitted atlas-env import
// list, so the caller (main.go) threads this in as a plain function value
// (service.TenantEnvironment) rather than the package importing atlas-env
// itself. Without it, decide() sees an empty or wrong ENVIRONMENT header and
// either fails open per FR-1.8 or is dropped by every consumer's ownership
// gate per FR-7.7.
func NewDecayTick(l logrus.FieldLogger, ctx context.Context, interval time.Duration, envContext func(context.Context) context.Context) *DecayTick {
	return &DecayTick{l: l, ctx: ctx, interval: interval, envContext: envContext}
}

func (r *DecayTick) SleepTime() time.Duration {
	return r.interval
}

func (r *DecayTick) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(r.ctx, "aran_combo_decay_tick")
	defer span.End()

	expired := GetMirror().ExpireIdle(time.Now())
	if len(expired) == 0 {
		return
	}
	processExpiries(r.l, ctx, expired, cancelComboBuff, r.envContext)
}

// cancelComboBuff drops the Combo Ability buff for one expired combo. The
// tenant-scoped context is built by processExpiries before this is called --
// the tick has no request context of its own, the same shape
// ProcessPoisonTicks uses in atlas-buffs.
func cancelComboBuff(l logrus.FieldLogger, ctx context.Context, e Expired) error {
	return buff.NewProcessor(l, ctx).Cancel(e.Field(), e.CharacterId(), int32(e.ComboId()))
}

// processExpiries cancels the Combo Ability buff for each expired combo and
// returns how many cancels succeeded. Failures are logged and swallowed:
// combo bookkeeping never fails a player action (task-217 NFR-4), and the
// next sweep will not retry because the count is already zero -- an orphaned
// buff icon is strictly better than a stalled tick.
//
// envContext must attach the environment that owns the tenant already on
// ctx (falling back to this pod's own, env.Self(), when the tenant is
// unknown) alongside the per-character tenant -- this is per-character
// lifecycle state driven by real gameplay, not one of the periodic
// multi-tenant background sweeps (session/task.go, session/processor.go,
// channel/task.go) that legitimately omit the environment. A nil envContext
// is a caller bug; tests exercise it directly since NewDecayTick's own
// tests can't observe the resulting context.
func processExpiries(l logrus.FieldLogger, ctx context.Context, expired []Expired, cancel func(l logrus.FieldLogger, ctx context.Context, e Expired) error, envContext func(context.Context) context.Context) int {
	n := 0
	for _, e := range expired {
		tctx := envContext(tenant.WithContext(ctx, e.Tenant()))
		if err := cancel(l, tctx, e); err != nil {
			l.WithError(err).Errorf("Aran combo: decay cancel emit failed for character [%d].", e.CharacterId())
			continue
		}
		l.Debugf("Aran combo: character [%d] decayed to zero; Combo Ability cancelled.", e.CharacterId())
		n++
	}
	return n
}
