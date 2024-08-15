package character

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/socket/model"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return requests.Provider[RestModel, Model](l)(requestById(l, span, tenant)(characterId), Extract)()
	}
}

func GetByIdWithInventory(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return requests.Provider[RestModel, Model](l)(requestByIdWithInventory(l, span, tenant)(characterId), Extract)()
	}
}

func Move(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, mm model.Movement) {
	moveCharacterCommandFunc := producer.ProviderImpl(l)(span)(EnvCommandTopicMovement)
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, mm model.Movement) {
		err := moveCharacterCommandFunc(move(tenant, worldId, channelId, mapId, characterId, mm))
		if err != nil {
			l.WithError(err).Errorf("Unable to distribute character movement to other services.")
		}
	}
}
