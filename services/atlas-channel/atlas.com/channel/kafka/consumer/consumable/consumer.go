package consumable

import (
	consumer2 "atlas-channel/kafka/consumer"
	monsterconsumer "atlas-channel/kafka/consumer/monster"
	consumable2 "atlas-channel/kafka/message/consumable"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	chatcb "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	petpkt "github.com/Chronicle20/atlas/libs/atlas-packet/pet/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("consumable_command")(consumable2.EnvEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(consumable2.EnvEventTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleScrollConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSkillBookResultEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRewardEffectConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRewardWonConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleVegaScrollConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleViciousHammerConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleCatchFailedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// errorAction is the client-facing reaction to a consumable ERROR event.
// Extracted from handleErrorConsumableEvent's if-chain so the routing is
// unit-testable without a server.Model, a writer.Producer, and the session
// registry -- which is what lets a test prove POTION_LOCKED is recognized
// rather than merely reaching the catch-all (task-280 FR-7).
type errorAction int

const (
	// actionUnstick sends StatChanged with an empty update list, releasing the
	// item-use exclusive-request lock and nothing else. This is the response
	// for POTION_LOCKED (no client message, PRD Q3), for CONSUME_FAILED, and
	// for any unrecognized type.
	actionUnstick errorAction = iota
	actionPetCashFoodError
	actionInventoryFull
	actionVegaInvalid
)

func consumableErrorAction(errorType string) errorAction {
	switch errorType {
	case consumable2.ErrorTypePetCannotConsume:
		return actionPetCashFoodError
	case consumable2.ErrorTypeInventoryFull:
		return actionInventoryFull
	case consumable2.ErrorTypeVegaInvalid:
		return actionVegaInvalid
	case consumable2.ErrorTypePotionLocked:
		// Explicit rather than left to the default: the default's response is
		// free to change when a future error type needs a different one, and
		// POTION_LOCKED must not silently inherit it.
		return actionUnstick
	default:
		return actionUnstick
	}
}

func handleErrorConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.ErrorBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.ErrorBody]) {
		if e.Type != consumable2.EventTypeError {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		sp := session.NewProcessor(l, ctx)
		unstick := session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)

		var err error
		switch consumableErrorAction(e.Body.Error) {
		case actionPetCashFoodError:
			err = sp.IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), session.Announce(l)(ctx)(wp)(petpkt.PetCashFoodResultWriter)(petpkt.NewPetCashFoodResultError().Encode))
		case actionInventoryFull:
			err = sp.IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
				if aerr := session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageDropPickUpInventoryFullBody())(s); aerr != nil {
					return aerr
				}
				return unstick(s)
			})
		case actionVegaInvalid:
			// INVALID (0x42 on both verified versions) closes the dialog with
			// the client's own "This item cannot be used." notice -- required,
			// since the dialog is excl-request-blocked after sending; then
			// enable-actions.
			err = sp.IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
				if verr := session.Announce(l)(ctx)(wp)(cashpkt.VegaScrollWriter)(cashpkt.VegaScrollInvalidBody())(s); verr != nil {
					return verr
				}
				return unstick(s)
			})
		default:
			// POTION_LOCKED, CONSUME_FAILED, and any unrecognized type: the
			// minimum bar is re-enabling client actions so the item-use
			// exclusive-request lock doesn't stay stuck.
			err = sp.IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), unstick)
		}
		if err != nil {
			l.WithError(err).Errorf("Unable to process error event for character [%d].", e.CharacterId)
		}
	}
}

func handleScrollConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.ScrollBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.ScrollBody]) {
		if e.Type != consumable2.EventTypeScroll {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
			return _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(), session.Announce(l)(ctx)(wp)(charcb.CharacterItemUpgradeWriter)(charcb.NewItemUpgrade(uint32(e.CharacterId), e.Body.Success, e.Body.Cursed, e.Body.LegendarySpirit, e.Body.WhiteScroll).Encode))
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to process scroll event for character [%d].", e.CharacterId)
		}
	}
}

// handleSkillBookResultEvent announces SKILL_LEARN_ITEM_RESULT for a
// skill-book outcome (task-125). canUse results broadcast to the whole map
// (the client renders the glow on the user's avatar for every observer and
// plays sound/message only locally); validation rejections and saga failures
// (canUse=false) go to the requester only — just enough to clear the client's
// exclusive-request lock.
func handleSkillBookResultEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.SkillBookResultBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.SkillBookResultBody]) {
		if e.Type != consumable2.EventTypeSkillBookResult {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		announce := session.Announce(l)(ctx)(wp)(charcb.CharacterSkillLearnItemResultWriter)(charcb.NewSkillLearnItemResult(uint32(e.CharacterId), e.Body.IsMasteryBook, e.Body.SkillId, e.Body.MasterLevel, e.Body.CanUse, e.Body.Success).Encode)
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
			if e.Body.CanUse {
				return _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(), announce)
			}
			return announce(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to process skill book result event for character [%d].", e.CharacterId)
		}
	}
}

func handleRewardEffectConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.RewardEffectBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.RewardEffectBody]) {
		if e.Type != consumable2.EventTypeRewardEffect {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
			// Self: the user sees the lottery-use effect.
			if aerr := session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterLotteryUseEffectBody(e.Body.BoxItemId, true, e.Body.Effect))(s); aerr != nil {
				l.WithError(aerr).Warnf("Unable to send lottery effect to character [%d].", e.CharacterId)
			}
			// Others in the map see the foreign effect.
			return _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterLotteryUseEffectForeignBody(s.CharacterId(), e.Body.BoxItemId, true, e.Body.Effect)))
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to process reward-effect event for character [%d].", e.CharacterId)
		}
	}
}

func handleRewardWonConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.RewardWonBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.RewardWonBody]) {
		if e.Type != consumable2.EventTypeRewardWon {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		sessions, err := session.NewProcessor(l, ctx).AllInChannelProvider(sc.WorldId(), sc.ChannelId())
		if err != nil {
			l.WithError(err).Error("Unable to get sessions for reward-won broadcast.")
			return
		}

		announceOp := session.Announce(l)(ctx)(wp)(chatcb.WorldMessageWriter)(writer.WorldMessageBlueTextItemBody("", "", e.Body.Message, e.Body.ItemId))
		for _, s := range sessions {
			if aerr := announceOp(s); aerr != nil {
				l.WithError(aerr).Warnf("Unable to send reward-won announce to session.")
			}
		}
	}
}

func handleVegaScrollConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.VegaScrollBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.VegaScrollBody]) {
		if e.Type != consumable2.EventTypeVegaScroll {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
			// Start + result back-to-back: the result is resolved immediately
			// server-side (owner decision); the client animates on its own
			// clock and latches the result byte.
			if err := session.Announce(l)(ctx)(wp)(cashpkt.VegaScrollWriter)(cashpkt.VegaScrollStartBody(e.Body.Success))(s); err != nil {
				return err
			}
			if err := session.Announce(l)(ctx)(wp)(cashpkt.VegaScrollWriter)(cashpkt.VegaScrollResultBody(e.Body.Success))(s); err != nil {
				return err
			}
			if err := _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(), session.Announce(l)(ctx)(wp)(charcb.CharacterItemUpgradeWriter)(charcb.NewItemUpgrade(uint32(e.CharacterId), e.Body.Success, e.Body.Cursed, false, false).Encode)); err != nil {
				return err
			}
			return session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to process vega scroll event for character [%d].", e.CharacterId)
		}
	}
}

func handleViciousHammerConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.ViciousHammerBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.ViciousHammerBody]) {
		if e.Type != consumable2.EventTypeViciousHammer {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		body := fieldpkt.ViciousHammerSuccessBody()
		if !e.Body.Success {
			body = fieldpkt.ViciousHammerFailureBody(fieldpkt.ViciousHammerFailureReason(e.Body.Reason))
		}
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), session.Announce(l)(ctx)(wp)(fieldcb.ViciousHammerWriter)(body))
		if err != nil {
			l.WithError(err).Errorf("Unable to process vicious hammer event for character [%d].", e.CharacterId)
		}
	}
}

// handleCatchFailedEvent renders a bridle-capture failure rejected before the
// request reached atlas-monsters (delay gate, inventory full, invalid item).
// It shares the wire-reason mapping and unlock idiom with the monster-side
// CATCH_FAILED handler via monsterconsumer.AnnounceCatchFailure so the two
// paths render identically.
func handleCatchFailedEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.CatchFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.CatchFailedBody]) {
		if e.Type != consumable2.EventTypeCatchFailed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		monsterconsumer.AnnounceCatchFailure(l, ctx, sc, wp, uint32(e.CharacterId), e.Body.ItemId, e.Body.Cause)
	}
}
