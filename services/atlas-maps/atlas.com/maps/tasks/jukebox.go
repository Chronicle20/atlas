package tasks

import (
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/map/jukebox"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const JukeboxTask = "jukebox_task"

type Jukebox struct {
	l          logrus.FieldLogger
	interval   time.Duration
	envContext func(context.Context) context.Context
}

func NewJukebox(l logrus.FieldLogger, interval time.Duration, envContext func(context.Context) context.Context) *Jukebox {
	return &Jukebox{l: l, interval: interval, envContext: envContext}
}

func (w *Jukebox) Run(ctx context.Context) {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(ctx, JukeboxTask)
	defer span.End()

	processExpiredJukebox(w.l, ctx, jukebox.GetExpired(), emitJukeboxEnd(w.l), w.envContext)
}

// emitJukeboxEnd produces the jukebox end event for one expired jukebox
// entry. Injected into processExpiredJukebox so the pure sweep logic can be
// tested with a spy in place of the real Kafka producer call.
func emitJukeboxEnd(l logrus.FieldLogger) func(ctx context.Context, e jukebox.ExpiredEntry) error {
	return func(ctx context.Context, e jukebox.ExpiredEntry) error {
		transactionId := uuid.New()
		f := e.Key.Field
		l.Debugf("Jukebox expired in map [%d] instance [%s]. Producing jukebox end event.", f.MapId(), f.Instance())
		return producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(jukebox.JukeboxEndEventProvider(transactionId, f, e.Entry.ItemId))
	}
}

// processExpiredJukebox originates this pod's own environment identity onto
// each expired jukebox entry's per-tenant context before emitting the end
// event and clearing the entry -- an empty ENVIRONMENT header would make
// decide() fail open per FR-1.8 and every live deployment, not just this
// pod's, would react to the jukebox end.
func processExpiredJukebox(l logrus.FieldLogger, ctx context.Context, expired []jukebox.ExpiredEntry, emit func(ctx context.Context, e jukebox.ExpiredEntry) error, envContext func(context.Context) context.Context) {
	for _, e := range expired {
		tctx := envContext(tenant.WithContext(ctx, e.Key.Tenant))
		if err := emit(tctx, e); err != nil {
			l.WithError(err).Errorf("Unable to produce jukebox end event for map [%d] instance [%s].", e.Key.Field.MapId(), e.Key.Field.Instance())
		}
		jukebox.DeleteEntry(e.Key)
	}
}

func (w *Jukebox) SleepTime() time.Duration {
	return w.interval
}
