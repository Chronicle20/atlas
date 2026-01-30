package note

import (
	consumer2 "atlas-notes/kafka/consumer"
	note2 "atlas-notes/kafka/message/note"
	"atlas-notes/note"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("note_command")(note2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(note2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleNoteCreate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleNoteDiscard(db))))
		}
	}
}

func handleNoteCreate(db *gorm.DB) message.Handler[note2.Command[note2.CommandCreateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c note2.Command[note2.CommandCreateBody]) {
		if c.Type != note2.CommandTypeCreate {
			return
		}

		// Call the processor to create the note
		_, _ = note.NewProcessor(l, ctx, db).CreateAndEmit(c.CharacterId, c.Body.SenderId, c.Body.Message, c.Body.Flag)
	}
}

func handleNoteDiscard(db *gorm.DB) message.Handler[note2.Command[note2.CommandDiscardBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c note2.Command[note2.CommandDiscardBody]) {
		if c.Type != note2.CommandTypeDiscard {
			return
		}

		// Call the processor to discard the notes
		_ = note.NewProcessor(l, ctx, db).DiscardAndEmit(c.WorldId, c.ChannelId, c.CharacterId, c.Body.NoteIds)
	}
}
