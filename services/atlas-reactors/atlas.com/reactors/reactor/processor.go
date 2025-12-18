package reactor

import (
	"atlas-reactors/kafka/producer"
	"atlas-reactors/reactor/data"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(id uint32) (Model, error) {
	return func(ctx context.Context) func(id uint32) (Model, error) {
		return func(id uint32) (Model, error) {
			return GetRegistry().Get(id)
		}
	}
}

func GetInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
			return GetRegistry().GetInMap(t, worldId, channelId, mapId), nil
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(b *ModelBuilder) error {
	return func(ctx context.Context) func(b *ModelBuilder) error {
		t := tenant.MustFromContext(ctx)
		return func(b *ModelBuilder) error {
			d, err := data.GetById(l)(ctx)(b.Classification())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve reactor [%d] game data.", b.Classification())
				return err
			}
			b.SetData(d)
			r := GetRegistry().Create(t, b)
			l.Debugf("Created reactor [%d] of [%d].", r.Id(), r.Classification())
			return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(createdStatusEventProvider(r))
		}
	}
}

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		ctx, span := otel.GetTracerProvider().Tracer("atlas-reactors").Start(context.Background(), "teardown")
		defer span.End()

		err := DestroyAll(l)(ctx)
		if err != nil {
			l.WithError(err).Errorf("Error destroying all reactors on teardown.")
		}
	}
}

func allByTenantProvider() model.Provider[map[tenant.Model][]Model] {
	return func() (map[tenant.Model][]Model, error) {
		return GetRegistry().GetAll(), nil
	}
}

func DestroyAll(l logrus.FieldLogger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return model.ForEachMap(allByTenantProvider(), DestroyInTenant(l)(ctx), model.ParallelExecute())
	}
}

func DestroyInTenant(l logrus.FieldLogger) func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
	return func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
		return func(t tenant.Model) model.Operator[[]Model] {
			return func(models []Model) error {
				tctx := tenant.WithContext(ctx, t)
				return model.ForEachSlice(model.FixedProvider(models), Destroy(l)(tctx), model.ParallelExecute())
			}
		}
	}
}

func Destroy(l logrus.FieldLogger) func(ctx context.Context) model.Operator[Model] {
	return func(ctx context.Context) model.Operator[Model] {
		return func(m Model) error {
			t := tenant.MustFromContext(ctx)
			GetRegistry().Remove(t, m.Id())
			return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(destroyedStatusEventProvider(m))
		}
	}
}
