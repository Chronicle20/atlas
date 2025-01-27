package thread

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, threadId uint32) (Model, error) {
	return func(ctx context.Context) func(guildId uint32, threadId uint32) (Model, error) {
		return func(guildId uint32, threadId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(guildId, threadId), Extract)()
		}
	}
}

func GetAll(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32) ([]Model, error) {
	return func(ctx context.Context) func(guildId uint32) ([]Model, error) {
		return func(guildId uint32) ([]Model, error) {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestAll(guildId), Extract, model.Filters[Model]())()
		}
	}
}

func ModifyThread(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, threadId uint32, notice bool, title string, message string, emoticonId uint32) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, threadId uint32, notice bool, title string, message string, emoticonId uint32) error {
		return func(guildId uint32, characterId uint32, threadId uint32, notice bool, title string, message string, emoticonId uint32) error {
			l.Debugf("Character [%d] is attempting to modify a guild thread [%d] to have, notice [%t], title [%s], message [%s], emoticonId [%d].", characterId, threadId, notice, title, message, emoticonId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(updateCommandProvider(guildId, characterId, threadId, notice, title, message, emoticonId))
		}
	}
}

func CreateThread(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, notice bool, title string, message string, emoticonId uint32) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, notice bool, title string, message string, emoticonId uint32) error {
		return func(guildId uint32, characterId uint32, notice bool, title string, message string, emoticonId uint32) error {
			l.Debugf("Character [%d] is attempting to create a guild thread to have, notice [%t], title [%s], message [%s], emoticonId [%d].", characterId, notice, title, message, emoticonId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(createCommandProvider(guildId, characterId, notice, title, message, emoticonId))
		}
	}
}

func DeleteThread(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, threadId uint32) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, threadId uint32) error {
		return func(guildId uint32, characterId uint32, threadId uint32) error {
			l.Debugf("Character [%d] attempting to delete guild thread [%d].", characterId, threadId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(deleteCommandProvider(guildId, characterId, threadId))
		}
	}
}

func ListThreads(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, startIndex uint32) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, startIndex uint32) error {
		return func(guildId uint32, characterId uint32, startIndex uint32) error {
			l.Debugf("Character [%d] attempting list threads starting at [%d].", characterId, startIndex)
			return nil
		}
	}
}

func ReplyToThread(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, threadId uint32, message string) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, threadId uint32, message string) error {
		return func(guildId uint32, characterId uint32, threadId uint32, message string) error {
			l.Debugf("Character [%d] attempting to reply to guild thread [%d] with message [%s].", characterId, threadId, message)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(addReplyCommandProvider(guildId, characterId, threadId, message))
		}
	}
}

func DeleteReply(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, threadId uint32, replyId uint32) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, threadId uint32, replyId uint32) error {
		return func(guildId uint32, characterId uint32, threadId uint32, replyId uint32) error {
			l.Debugf("Character [%d] attempting to delete reply [%d] in guild thread [%d].", characterId, replyId, threadId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(deleteReplyCommandProvider(guildId, characterId, threadId, replyId))
		}
	}
}
