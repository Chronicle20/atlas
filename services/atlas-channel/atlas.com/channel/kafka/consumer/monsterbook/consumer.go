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

// Package-level seams: these wrap the singleton/registry-backed
// session and map plumbing so unit tests can swap them out without
// standing up a session registry. Production code paths use the
// real values defined here; consumer_test.go replaces them via
// the helpers below.
var (
	// sessionForCharacter resolves a session for the given characterId
	// on the channel and invokes f if one exists. Returns nil if no
	// session is present (matching session.Processor.IfPresentByCharacterId).
	sessionForCharacter = func(l logrus.FieldLogger, ctx context.Context, sc server.Model, characterId uint32, f model.Operator[session.Model]) {
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, f)
	}

	// announceSetCard writes a clientbound MonsterBookSetCard to the owner.
	announceSetCard = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, body mbcb.SetCard) error {
		return session.Announce(l)(ctx)(wp)(mbcb.MonsterBookSetCardWriter)(body.Encode)(s)
	}

	// announceCardGetEffect writes the owner-visible CardGet effect.
	announceCardGetEffect = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) error {
		return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterMonsterBookCardGetEffectBody())(s)
	}

	// broadcastCardGetEffectForeign fans the foreign-effect packet out to
	// every other session in the owner's map.
	broadcastCardGetEffectForeign = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) {
		_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterMonsterBookCardGetEffectForeignBody(s.CharacterId())))
	}

	// announceSetCover writes a clientbound MonsterBookSetCover to the owner.
	announceSetCover = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, body mbcb.SetCover) error {
		return session.Announce(l)(ctx)(wp)(mbcb.MonsterBookSetCoverWriter)(body.Encode)(s)
	}
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

		sessionForCharacter(l, ctx, sc, e.CharacterId, func(s session.Model) error {
			// Always send the SetCard inventory mutation to the owner.
			if err := announceSetCard(l, ctx, wp, s, mbcb.SetCard{
				CardId: e.Body.CardId,
				Level:  e.Body.NewLevel,
				Added:  true,
			}); err != nil {
				l.WithError(err).Errorf("Unable to send MonsterBookSetCard for character [%d] card [%d].", e.CharacterId, e.Body.CardId)
			}

			// Only emit the visual "got a card" effect when the card was
			// not already at the max level in the player's collection.
			if !e.Body.Full {
				if err := announceCardGetEffect(l, ctx, wp, s); err != nil {
					l.WithError(err).Errorf("Unable to send MonsterBookCardGet effect for character [%d].", e.CharacterId)
				}

				broadcastCardGetEffectForeign(l, ctx, wp, s)
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

		sessionForCharacter(l, ctx, sc, e.CharacterId, func(s session.Model) error {
			if err := announceSetCover(l, ctx, wp, s, mbcb.SetCover{
				CardId: e.Body.CoverCardId,
			}); err != nil {
				l.WithError(err).Errorf("Unable to send MonsterBookSetCover for character [%d] card [%d].", e.CharacterId, e.Body.CoverCardId)
			}
			return nil
		})
	}
}
