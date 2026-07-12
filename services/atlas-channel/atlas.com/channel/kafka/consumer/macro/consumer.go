package macro

import (
	consumer2 "atlas-channel/kafka/consumer"
	macro2 "atlas-channel/kafka/message/macro"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"sort"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("skill_macro_status_event")(macro2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(macro2.EnvStatusEventTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleUpdated(sc server.Model, wp writer.Producer) message.Handler[macro2.StatusEvent[macro2.StatusEventUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e macro2.StatusEvent[macro2.StatusEventUpdatedBody]) {
		if e.Type != macro2.StatusEventTypeUpdated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, announceMacros(l)(ctx)(wp)(e.Body.Macros))
		if err != nil {
			l.WithError(err).Errorf("Unable to update skill macros for character [%d].", e.CharacterId)
		}
	}
}

func announceMacros(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(bodies []macro2.MacroBody) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(bodies []macro2.MacroBody) model.Operator[session.Model] {
		return func(wp writer.Producer) func(bodies []macro2.MacroBody) model.Operator[session.Model] {
			return func(bodies []macro2.MacroBody) model.Operator[session.Model] {
				sorted := make([]macro2.MacroBody, len(bodies))
				copy(sorted, bodies)
				sort.Slice(sorted, func(i, j int) bool {
					return sorted[i].Id < sorted[j].Id
				})
				mms := make([]packetmodel.Macro, 0, len(sorted))
				for _, sm := range sorted {
					mms = append(mms, packetmodel.NewMacro(sm.Name, sm.Shout, skill2.Id(sm.SkillId1), skill2.Id(sm.SkillId2), skill2.Id(sm.SkillId3)))
				}
				macros := packetmodel.NewMacros(mms...)
				return session.Announce(l)(ctx)(wp)(charpkt.CharacterSkillMacroWriter)(macros.Encode)
			}
		}
	}
}
