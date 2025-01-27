package thread

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func statusEventCreatedProvider(worldId byte, guildId uint32, threadId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[createdStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		Type:     StatusEventTypeCreated,
		Body:     createdStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventUpdatedProvider(worldId byte, guildId uint32, threadId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[updatedStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		Type:     StatusEventTypeUpdated,
		Body:     updatedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventDeletedProvider(worldId byte, guildId uint32, threadId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[deletedStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		Type:     StatusEventTypeDeleted,
		Body:     deletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventReplyAddedProvider(worldId byte, guildId uint32, threadId uint32, replyId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[replyAddedStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		Type:     StatusEventTypeReplyAdded,
		Body: replyAddedStatusEventBody{
			ReplyId: replyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventReplyDeletedProvider(worldId byte, guildId uint32, threadId uint32, replyId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[replyDeletedStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		Type:     StatusEventTypeReplyDeleted,
		Body: replyDeletedStatusEventBody{
			ReplyId: replyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
