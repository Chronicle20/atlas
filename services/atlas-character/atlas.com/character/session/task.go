package session

import (
	"atlas-character/character"
	"atlas-character/session/history"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const TimeoutTask = "timeout"

type Timeout struct {
	l          logrus.FieldLogger
	db         *gorm.DB
	interval   time.Duration
	timeout    time.Duration
	envContext func(context.Context) context.Context
}

// NewTimeout builds the periodic session-timeout sweep. envContext originates
// the environment that owns the session's tenant, falling back to this
// pod's own environment identity when the tenant's environment cannot be
// resolved, onto each timed-out session's per-character context before
// LogoutAndEmit produces a real Kafka event -- session is outside
// env-domain-guard's permitted atlas-env import list (main.go, kafka/,
// rest/, socket/), so the caller (main.go) threads this in as a plain
// function value rather than the package importing atlas-env itself.
// Without it, the logout event would carry either an empty ENVIRONMENT
// header (fails decide() open per FR-1.8: every live deployment, not just
// this pod's, would act on the logout) or the wrong pod's environment (a
// sparse environment's tenant served by a baseline pod would have its
// logout dropped at every consumer's ownership gate, FR-7.7).
func NewTimeout(l logrus.FieldLogger, db *gorm.DB, interval time.Duration, envContext func(context.Context) context.Context) *Timeout {
	timeout := time.Duration(5000) * time.Millisecond
	l.Infof("Initializing timeout task to run every %dms, timeout transition session older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, db, interval, timeout, envContext}
}

func (t *Timeout) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-character").Start(context.Background(), TimeoutTask)
	defer span.End()

	cur := time.Now()

	t.l.Debugf("Executing timeout task.")
	cs := GetRegistry().GetAll(sctx)
	for _, m := range cs {
		tctx := t.sessionTenantContext(sctx, m)
		cp := character.NewProcessor(t.l, tctx, t.db)
		cha := channel.NewModel(m.WorldId(), m.ChannelId())

		if m.State() == StateTransition && cur.Sub(m.Age()) > t.timeout {
			t.l.Debugf("Timing out record for character [%d].", m.CharacterId())
			GetRegistry().Remove(tctx, m.CharacterId())

			// Close session history
			hp := history.NewProcessor(t.l, tctx, t.db)
			err := hp.EndSession(m.CharacterId())
			if err != nil {
				t.l.WithError(err).Warnf("Failed to end session history for character [%d].", m.CharacterId())
			}

			err = cp.LogoutAndEmit(uuid.New(), m.CharacterId(), cha)
			if err != nil {
				t.l.WithError(err).Errorf("Unable to logout character [%d] as a result of session being destroyed.", m.CharacterId())
			}
		}
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}

// sessionTenantContext builds the per-character context that LogoutAndEmit and
// history.EndSession run under: the session's tenant, then envContext to
// originate the environment that owns that tenant on top. Extracted so the
// origination itself is directly testable without standing up a DB or the
// Redis-backed registry that Run's other callers require.
func (t *Timeout) sessionTenantContext(sctx context.Context, m Model) context.Context {
	return t.envContext(tenant.WithContext(sctx, m.Tenant()))
}
