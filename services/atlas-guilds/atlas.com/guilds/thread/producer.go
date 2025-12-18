package thread

import (
	thread2 "atlas-guilds/kafka/message/thread"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func statusEventCreatedProvider(worldId byte, guildId uint32, threadId uint32, actorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.StatusEvent[thread2.CreatedStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		ActorId:  actorId,
		Type:     thread2.StatusEventTypeCreated,
		Body:     thread2.CreatedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventUpdatedProvider(worldId byte, guildId uint32, threadId uint32, actorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.StatusEvent[thread2.UpdatedStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		ActorId:  actorId,
		Type:     thread2.StatusEventTypeUpdated,
		Body:     thread2.UpdatedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventDeletedProvider(worldId byte, guildId uint32, threadId uint32, actorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.StatusEvent[thread2.DeletedStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		ActorId:  actorId,
		Type:     thread2.StatusEventTypeDeleted,
		Body:     thread2.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventReplyAddedProvider(worldId byte, guildId uint32, threadId uint32, actorId uint32, replyId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.StatusEvent[thread2.ReplyAddedStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		ActorId:  actorId,
		Type:     thread2.StatusEventTypeReplyAdded,
		Body: thread2.ReplyAddedStatusEventBody{
			ReplyId: replyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventReplyDeletedProvider(worldId byte, guildId uint32, threadId uint32, actorId uint32, replyId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.StatusEvent[thread2.ReplyDeletedStatusEventBody]{
		WorldId:  worldId,
		GuildId:  guildId,
		ThreadId: threadId,
		ActorId:  actorId,
		Type:     thread2.StatusEventTypeReplyDeleted,
		Body: thread2.ReplyDeletedStatusEventBody{
			ReplyId: replyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
