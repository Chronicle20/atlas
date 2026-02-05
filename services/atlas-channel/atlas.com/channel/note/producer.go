package note

import (
	"atlas-channel/kafka/message/note"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(ch channel.Model, actorId uint32, receiverId uint32, message string, flag byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &note.Command[note.CommandCreateBody]{
		WorldId:     ch.WorldId(),
		ChannelId:   ch.Id(),
		CharacterId: receiverId,
		Type:        note.CommandTypeCreate,
		Body: note.CommandCreateBody{
			SenderId: actorId,
			Message:  message,
			Flag:     flag,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DiscardCommandProvider(ch channel.Model, characterId uint32, noteIds []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &note.Command[note.CommandDiscardBody]{
		WorldId:     ch.WorldId(),
		ChannelId:   ch.Id(),
		CharacterId: characterId,
		Type:        note.CommandTypeDiscard,
		Body: note.CommandDiscardBody{
			NoteIds: noteIds,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
