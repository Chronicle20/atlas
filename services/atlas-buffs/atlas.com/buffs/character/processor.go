package character

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func GetById(ctx context.Context) func(characterId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	return func(characterId uint32) (Model, error) {
		return GetRegistry().Get(t, characterId)
	}
}

func Apply(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, characterId uint32, sourceId uint32, duration int32, changes []stat.Model) error {
	return func(ctx context.Context) func(worldId byte, characterId uint32, sourceId uint32, duration int32, changes []stat.Model) error {
		t := tenant.MustFromContext(ctx)
		return func(worldId byte, characterId uint32, sourceId uint32, duration int32, changes []stat.Model) error {
			_ = GetRegistry().Apply(t, worldId, characterId, sourceId, duration, changes)
			_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(appliedStatusEventProvider(worldId, characterId, sourceId, duration, changes))
			return nil
		}
	}
}

func Cancel(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, characterId uint32, sourceId uint32) error {
	return func(ctx context.Context) func(worldId byte, characterId uint32, sourceId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(worldId byte, characterId uint32, sourceId uint32) error {
			b, err := GetRegistry().Cancel(t, characterId, sourceId)
			if errors.Is(err, ErrNotFound) {
				return nil
			}
			_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Duration(), b.Changes()))
			return nil
		}
	}
}

func ExpireBuffs(l logrus.FieldLogger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		ts, err := GetRegistry().GetTenants()
		if err != nil {
			return err
		}

		for _, t := range ts {
			go func() {
				tctx := tenant.WithContext(ctx, t)
				for _, c := range GetRegistry().GetCharacters(t) {
					ebs := GetRegistry().GetExpired(t, c.Id())
					for _, eb := range ebs {
						l.Debugf("Expired buff for character [%d] from [%d].", c.Id(), eb.SourceId())
						_ = producer.ProviderImpl(l)(tctx)(EnvEventStatusTopic)(expiredStatusEventProvider(c.WorldId(), c.Id(), eb.SourceId(), eb.Duration(), eb.Changes()))
					}
				}
			}()
		}
		return nil
	}
}
