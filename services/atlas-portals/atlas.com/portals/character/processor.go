package character

import (
	"atlas-portals/kafka/producer"
	"atlas-portals/tenant"
	"context"
	"github.com/sirupsen/logrus"
)

func EnableActions(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, characterId uint32) {
	return func(worldId byte, channelId byte, characterId uint32) {
		_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(enableActionsProvider(tenant, worldId, channelId, characterId))
	}
}
