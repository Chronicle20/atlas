package channel

import (
	"atlas-login/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"math/rand"
)

func byIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte) model.Provider[Model] {
	return func(worldId byte, channelId byte) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestChannel(l, span, tenant)(worldId, channelId), Extract)
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte) (Model, error) {
	return func(worldId byte, channelId byte) (Model, error) {
		return byIdModelProvider(l, span, tenant)(worldId, channelId)()
	}
}

func byWorldModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte) model.Provider[[]Model] {
	return func(worldId byte) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestChannels(l, span, tenant)(worldId), Extract)
	}
}

func GetForWorld(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte) ([]Model, error) {
	return func(worldId byte) ([]Model, error) {
		return byWorldModelProvider(l, span, tenant)(worldId)()
	}
}

func GetRandomInWorld(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte) (Model, error) {
	return func(worldId byte) (Model, error) {
		cs, err := GetForWorld(l, span, tenant)(worldId)
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
