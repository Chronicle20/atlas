package thread

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(guildId uint32, characterId uint32, notice bool, title string, message string, emoticonId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &command[createCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        CommandTypeCreate,
		Body: createCommandBody{
			Notice:     notice,
			Title:      title,
			Message:    message,
			EmoticonId: emoticonId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func updateCommandProvider(guildId uint32, characterId uint32, threadId uint32, notice bool, title string, message string, emoticonId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &command[updateCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        CommandTypeUpdate,
		Body: updateCommandBody{
			ThreadId:   threadId,
			Notice:     notice,
			Title:      title,
			Message:    message,
			EmoticonId: emoticonId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func deleteCommandProvider(guildId uint32, characterId uint32, threadId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &command[deleteCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        CommandTypeDelete,
		Body: deleteCommandBody{
			ThreadId: threadId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func addReplyCommandProvider(guildId uint32, characterId uint32, threadId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &command[addReplyCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        CommandTypeAddReply,
		Body: addReplyCommandBody{
			ThreadId: threadId,
			Message:  message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func deleteReplyCommandProvider(guildId uint32, characterId uint32, threadId uint32, replyId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &command[deleteReplyCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        CommandTypeDeleteReply,
		Body: deleteReplyCommandBody{
			ThreadId: threadId,
			ReplyId:  replyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
