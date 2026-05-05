package monsterbook

import (
	consumer2 "atlas-channel/kafka/consumer"
	mbmsg "atlas-channel/kafka/message/monsterbook"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	mbcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound/monsterbook"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// InitConsumers registers a single consumer config for the monster book
// status topic. The same topic backs CARD_ADDED, COVER_CHANGED and
// STATS_CHANGED events; per-type handlers filter on Type.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_book_status_event")(mbmsg.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

// InitHandlers registers the per-type handlers on the monster book status
// topic for the given server channel.
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(mbmsg.EnvEventTopicStatus)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCardAdded(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCoverChanged(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

// handleCardAdded fans out a SetCard packet to the owner (always) plus the
// CardGet effect to the owner and a foreign effect to other players in the
// owner's map (only when the card is not yet at full level).
func handleCardAdded(sc server.Model, wp writer.Producer) message.Handler[mbmsg.StatusEvent[mbmsg.CardAddedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mbmsg.StatusEvent[mbmsg.CardAddedBody]) {
		if e.Type != mbmsg.StatusEventTypeCardAdded {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			// Always send the SetCard inventory mutation to the owner.
			if err := session.Announce(l)(ctx)(wp)(mbcb.MonsterBookSetCardWriter)(mbcb.SetCard{
				CardId: e.Body.CardId,
				Level:  e.Body.NewLevel,
				Added:  true,
			}.Encode)(s); err != nil {
				l.WithError(err).Errorf("Unable to send MonsterBookSetCard for character [%d] card [%d].", e.CharacterId, e.Body.CardId)
			}

			// Only emit the visual "got a card" effect when the card was
			// not already at the max level in the player's collection.
			if !e.Body.Full {
				if err := session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterMonsterBookCardGetEffectBody())(s); err != nil {
					l.WithError(err).Errorf("Unable to send MonsterBookCardGet effect for character [%d].", e.CharacterId)
				}

				_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterMonsterBookCardGetEffectForeignBody(s.CharacterId())))
			}

			return nil
		})
	}
}

// handleCoverChanged sends an authoritative SetCover packet to the owner so
// the cover image in the monster book UI updates immediately.
func handleCoverChanged(sc server.Model, wp writer.Producer) message.Handler[mbmsg.StatusEvent[mbmsg.CoverChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mbmsg.StatusEvent[mbmsg.CoverChangedBody]) {
		if e.Type != mbmsg.StatusEventTypeCoverChanged {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			if err := session.Announce(l)(ctx)(wp)(mbcb.MonsterBookSetCoverWriter)(mbcb.SetCover{
				CardId: e.Body.CoverCardId,
			}.Encode)(s); err != nil {
				l.WithError(err).Errorf("Unable to send MonsterBookSetCover for character [%d] card [%d].", e.CharacterId, e.Body.CoverCardId)
			}
			return nil
		})
	}
}
