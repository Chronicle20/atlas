package tasks

import (
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/producer"
	"atlas-maps/map/weather"
	"context"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const WeatherTask = "weather_task"

type Weather struct {
	l        logrus.FieldLogger
	interval time.Duration
}

func NewWeather(l logrus.FieldLogger, interval time.Duration) *Weather {
	return &Weather{l: l, interval: interval}
}

func (w *Weather) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), WeatherTask)
	defer span.End()

	expired := weather.GetExpired()
	for _, e := range expired {
		tctx := tenant.WithContext(ctx, e.Key.Tenant)
		transactionId := uuid.New()
		f := e.Key.Field

		w.l.Debugf("Weather expired in map [%d] instance [%s]. Producing weather end event.", f.MapId(), f.Instance())

		err := producer.ProviderImpl(w.l)(tctx)(mapKafka.EnvEventTopicMapStatus)(weather.WeatherEndEventProvider(transactionId, f, e.Entry.ItemId))
		if err != nil {
			w.l.WithError(err).Errorf("Unable to produce weather end event for map [%d] instance [%s].", f.MapId(), f.Instance())
		}

		weather.DeleteEntry(e.Key)
	}
}

func (w *Weather) SleepTime() time.Duration {
	return w.interval
}
