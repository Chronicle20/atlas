package channel

import (
	"atlas-login/tenant"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"math/rand"
)

func byIdModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte) model.Provider[Model] {
	return func(worldId byte, channelId byte) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestChannel(ctx, tenant)(worldId, channelId), Extract)
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte) (Model, error) {
	return func(worldId byte, channelId byte) (Model, error) {
		return byIdModelProvider(l, ctx, tenant)(worldId, channelId)()
	}
}

func byWorldModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte) model.Provider[[]Model] {
	return func(worldId byte) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestChannels(ctx, tenant)(worldId), Extract)
	}
}

func GetForWorld(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte) ([]Model, error) {
	return func(worldId byte) ([]Model, error) {
		return byWorldModelProvider(l, ctx, tenant)(worldId)()
	}
}

func GetRandomInWorld(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte) (Model, error) {
	return func(worldId byte) (Model, error) {
		cs, err := GetForWorld(l, ctx, tenant)(worldId)
		if err != nil {
			return Model{}, err
		}
		if len(cs) == 0 {
			return Model{}, errors.New("no channel found")
		}

		ri := rand.Intn(len(cs))
		return cs[ri], nil
	}
}
