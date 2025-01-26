package character

import (
	"atlas-fame/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)()
		}
	}
}

func RequestChangeFame(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId byte, actorId uint32, amount int8) error {
	return func(ctx context.Context) func(characterId uint32, worldId byte, actorId uint32, amount int8) error {
		return func(characterId uint32, worldId byte, actorId uint32, amount int8) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestChangeFameCommandProvider(characterId, worldId, actorId, amount))
		}
	}
}
