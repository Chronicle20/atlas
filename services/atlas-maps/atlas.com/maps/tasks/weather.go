package tasks

import (
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/map/weather"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const WeatherTask = "weather_task"

type Weather struct {
	l          logrus.FieldLogger
	interval   time.Duration
	envContext func(context.Context) context.Context
}

func NewWeather(l logrus.FieldLogger, interval time.Duration, envContext func(context.Context) context.Context) *Weather {
	return &Weather{l: l, interval: interval, envContext: envContext}
}

func (w *Weather) Run(ctx context.Context) {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(ctx, WeatherTask)
	defer span.End()

	processExpiredWeather(w.l, ctx, weather.GetExpired(), emitWeatherEnd(w.l), w.envContext)
}

// emitWeatherEnd produces the weather end event for one expired weather
// entry. Injected into processExpiredWeather so the pure sweep logic can be
// tested with a spy in place of the real Kafka producer call.
func emitWeatherEnd(l logrus.FieldLogger) func(ctx context.Context, e weather.ExpiredEntry) error {
	return func(ctx context.Context, e weather.ExpiredEntry) error {
		transactionId := uuid.New()
		f := e.Key.Field
		l.Debugf("Weather expired in map [%d] instance [%s]. Producing weather end event.", f.MapId(), f.Instance())
		return producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(weather.WeatherEndEventProvider(transactionId, f, e.Entry.ItemId))
	}
}

// processExpiredWeather originates this pod's own environment identity onto
// each expired weather entry's per-tenant context before emitting the end
// event and clearing the entry -- an empty ENVIRONMENT header would make
// decide() fail open per FR-1.8 and every live deployment, not just this
// pod's, would react to the weather end.
func processExpiredWeather(l logrus.FieldLogger, ctx context.Context, expired []weather.ExpiredEntry, emit func(ctx context.Context, e weather.ExpiredEntry) error, envContext func(context.Context) context.Context) {
	for _, e := range expired {
		tctx := envContext(tenant.WithContext(ctx, e.Key.Tenant))
		if err := emit(tctx, e); err != nil {
			l.WithError(err).Errorf("Unable to produce weather end event for map [%d] instance [%s].", e.Key.Field.MapId(), e.Key.Field.Instance())
		}
		weather.DeleteEntry(e.Key)
	}
}

func (w *Weather) SleepTime() time.Duration {
	return w.interval
}
