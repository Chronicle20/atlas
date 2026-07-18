package berserk

import (
	character2 "atlas-buffs/kafka/message/character"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func berserkStatusEventProvider(transactionId uuid.UUID, m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.CharacterId()))
	value := &character2.StatusEvent[character2.BerserkStatusEventBody]{
		WorldId:     m.WorldId(),
		CharacterId: m.CharacterId(),
		Type:        character2.EventStatusTypeBerserk,
		Body: character2.BerserkStatusEventBody{
			TransactionId:  transactionId,
			ChannelId:      m.ChannelId(),
			SkillId:        uint32(skill.DarkKnightBerserkId),
			CharacterLevel: m.CharacterLevel(),
			SkillLevel:     m.SkillLevel(),
			Active:         m.Active(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
