package note

import (
	consumer2 "atlas-channel/kafka/consumer"
	note2 "atlas-channel/kafka/message/note"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	notecb "github.com/Chronicle20/atlas/libs/atlas-packet/note/clientbound"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("note_status_event")(note2.EnvEventTopicNoteStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(note2.EnvEventTopicNoteStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleNoteCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleNoteCreated(sc server.Model, wp writer.Producer) message.Handler[note2.StatusEvent[note2.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e note2.StatusEvent[note2.StatusEventCreatedBody]) {
		if e.Type != note2.StatusEventTypeCreated {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		// Process the note created event
		l.Debugf("Processing note created event for character [%d], note ID [%d]", e.CharacterId, e.Body.NoteId)

		// Get the session for the character
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, processNoteCreated(l)(ctx)(wp)(e.Body))
	}
}

func processNoteCreated(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(body note2.StatusEventCreatedBody) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(body note2.StatusEventCreatedBody) model.Operator[session.Model] {
		return func(wp writer.Producer) func(body note2.StatusEventCreatedBody) model.Operator[session.Model] {
			return func(body note2.StatusEventCreatedBody) model.Operator[session.Model] {
				return func(s session.Model) error {
					// Send the note to the client
					err := session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteRefreshBody())(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to display note [%d] for character [%d]", body.NoteId, s.CharacterId())
					}
					return nil
				}
			}
		}
	}
}
