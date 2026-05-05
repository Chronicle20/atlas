package monsterbook

import (
	mbmsg "atlas-channel/kafka/message/monsterbook"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// SetCoverCommandProvider builds the provider for a SET_COVER monster book command.
// CharacterId / CoverCardId stay as wire-primitive uint32 in the kafka envelope; we
// cast at the boundary.
func SetCoverCommandProvider(tenantId uuid.UUID, characterId character.Id, coverCardId item.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mbmsg.Command[mbmsg.SetCoverBody]{
		TenantId:    tenantId,
		CharacterId: uint32(characterId),
		EventId:     uuid.New(),
		Type:        mbmsg.CommandTypeSetCover,
		Body: mbmsg.SetCoverBody{
			CoverCardId: uint32(coverCardId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
