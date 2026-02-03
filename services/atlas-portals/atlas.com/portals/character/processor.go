package character

import (
	"atlas-portals/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

func EnableActions(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, characterId uint32) {
	return func(ctx context.Context) func(f field.Model, characterId uint32) {
		return func(f field.Model, characterId uint32) {
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(enableActionsProvider(f, characterId))
		}
	}
}
