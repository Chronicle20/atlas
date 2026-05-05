package monsterbook

import (
	mbmsg "atlas-channel/kafka/message/monsterbook"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// SetCoverCommandProvider builds the provider for a SET_COVER monster book command.
func SetCoverCommandProvider(tenantId uuid.UUID, characterId uint32, coverCardId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mbmsg.Command[mbmsg.SetCoverBody]{
		TenantId:    tenantId,
		CharacterId: characterId,
		EventId:     uuid.New(),
		Type:        mbmsg.CommandTypeSetCover,
		Body: mbmsg.SetCoverBody{
			CoverCardId: coverCardId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
