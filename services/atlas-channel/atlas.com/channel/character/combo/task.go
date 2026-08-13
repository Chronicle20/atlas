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
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewDecayTick(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *DecayTick {
	return &DecayTick{l: l, ctx: ctx, interval: interval}
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
	processExpiries(r.l, ctx, expired, cancelComboBuff)
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
func processExpiries(l logrus.FieldLogger, ctx context.Context, expired []Expired, cancel func(l logrus.FieldLogger, ctx context.Context, e Expired) error) int {
	n := 0
	for _, e := range expired {
		tctx := tenant.WithContext(ctx, e.Tenant())
		if err := cancel(l, tctx, e); err != nil {
			l.WithError(err).Errorf("Aran combo: decay cancel emit failed for character [%d].", e.CharacterId())
			continue
		}
		l.Debugf("Aran combo: character [%d] decayed to zero; Combo Ability cancelled.", e.CharacterId())
		n++
	}
	return n
}
