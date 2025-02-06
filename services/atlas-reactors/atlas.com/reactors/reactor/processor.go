package reactor

import (
	"atlas-reactors/kafka/producer"
	"atlas-reactors/reactor/data"
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
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
