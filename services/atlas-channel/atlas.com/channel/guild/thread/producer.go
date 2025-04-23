package thread

import (
	thread2 "atlas-channel/kafka/message/guild/thread"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(guildId uint32, characterId uint32, notice bool, title string, message string, emoticonId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.Command[thread2.CreateCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        thread2.CommandTypeCreate,
		Body: thread2.CreateCommandBody{
			Notice:     notice,
			Title:      title,
			Message:    message,
			EmoticonId: emoticonId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func UpdateCommandProvider(guildId uint32, characterId uint32, threadId uint32, notice bool, title string, message string, emoticonId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.Command[thread2.UpdateCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        thread2.CommandTypeUpdate,
		Body: thread2.UpdateCommandBody{
			ThreadId:   threadId,
			Notice:     notice,
			Title:      title,
			Message:    message,
			EmoticonId: emoticonId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeleteCommandProvider(guildId uint32, characterId uint32, threadId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.Command[thread2.DeleteCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        thread2.CommandTypeDelete,
		Body: thread2.DeleteCommandBody{
			ThreadId: threadId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AddReplyCommandProvider(guildId uint32, characterId uint32, threadId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.Command[thread2.AddReplyCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        thread2.CommandTypeAddReply,
		Body: thread2.AddReplyCommandBody{
			ThreadId: threadId,
			Message:  message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeleteReplyCommandProvider(guildId uint32, characterId uint32, threadId uint32, replyId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &thread2.Command[thread2.DeleteReplyCommandBody]{
		GuildId:     guildId,
		CharacterId: characterId,
		Type:        thread2.CommandTypeDeleteReply,
		Body: thread2.DeleteReplyCommandBody{
			ThreadId: threadId,
			ReplyId:  replyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
