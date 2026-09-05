package mount

import (
	mountmessage "atlas-mounts/kafka/message"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const TirednessTaskName = "tiredness"

// getActive is a function seam over the active-mount registry. Production wires
// it to the real Redis-backed registry; tests override it to supply a fixed set
// of active entries without touching Redis.
var getActive = func(ctx context.Context) ([]ActiveEntry, error) {
	return GetRegistry().GetActive(ctx)
}

// applyTick is a function seam over the per-entry tick path. Production builds a
// tenant-scoped processor and emits the TICK status event through a buffer;
// tests override it to record the (tenant-via-ctx, worldId, characterId) it was
// called with, exercising the loop without a database or Kafka.
var applyTick = func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, worldId world.Id, characterId uint32) error {
	return database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
		p := NewProcessor(l, ctx, tx)
		return mountmessage.Emit(outbox.EmitProvider(l, ctx, tx))(func(mb *mountmessage.Buffer) error {
			return p.ApplyTick(mb)(worldId, characterId)
		})
	})
}

// TirednessTask is the 60-second ticker that increments tiredness on every
// active (tamed) mount. Skill-only mounts are never present in the registry, so
// they are never ticked (FR-2.2). A single task iterates the registry; there are
// no per-character goroutines or timers.
type TirednessTask struct {
	l          logrus.FieldLogger
	db         *gorm.DB
	interval   time.Duration
	envContext func(context.Context) context.Context
}

// NewTirednessTask constructs the tiredness ticker. envContext originates
// this pod's own environment identity onto each active entry's per-tenant
// context before it reaches the outbox emit in applyTick -- see 99c0e598d /
// task-232.
func NewTirednessTask(l logrus.FieldLogger, db *gorm.DB, interval time.Duration, envContext func(context.Context) context.Context) *TirednessTask {
	l.Infof("Initializing %s task to run every %dms", TirednessTaskName, interval.Milliseconds())
	return &TirednessTask{l: l, db: db, interval: interval, envContext: envContext}
}

func (t *TirednessTask) Run(ctx context.Context) {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-mounts").Start(ctx, TirednessTaskName)
	defer span.End()

	t.l.Debugf("Executing %s task.", TirednessTaskName)
	entries, err := getActive(sctx)
	if err != nil {
		t.l.WithError(err).Errorf("Unable to enumerate active mounts for %s task.", TirednessTaskName)
		return
	}

	processActive(t.l, sctx, t.db, entries, applyTick, t.envContext)
}

// processActive is the pure per-entry sweep extracted from Run() so the
// envContext threading can be pinned without a database or Kafka. The
// registry spans all tenants and the ticker has no ambient tenant, so each
// entry carries its own tenant; tick originates this pod's environment
// identity onto that tenant-scoped context before calling tick.
func processActive(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, entries []ActiveEntry, tick func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, worldId world.Id, characterId uint32) error, envContext func(context.Context) context.Context) {
	for _, e := range entries {
		tctx := envContext(tenant.WithContext(ctx, e.Tenant))
		if err := tick(l, tctx, db, e.Ctx.WorldId, e.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to apply tiredness tick for character [%d].", e.CharacterId)
		}
	}
}

func (t *TirednessTask) SleepTime() time.Duration {
	return t.interval
}
