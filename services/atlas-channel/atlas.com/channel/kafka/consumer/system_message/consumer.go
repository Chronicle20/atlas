package system_message

import (
	consumer2 "atlas-channel/kafka/consumer"
	system_message2 "atlas-channel/kafka/message/system_message"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("system_message_command")(system_message2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(system_message2.EnvCommandTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSendMessage(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handlePlayPortalSound(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleShowInfo(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleShowInfoText(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdateAreaInfo(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleShowHint(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleShowGuideHint(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleShowIntro(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleFieldEffect(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUiLock(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUiDisable(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleSendMessage(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.SendMessageBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.SendMessageBody]) {
		if cmd.Type != system_message2.CommandSendMessage {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		// Map message type to body producer
		var bodyProducer writer.BodyProducer
		switch cmd.Body.MessageType {
		case "NOTICE":
			bodyProducer = writer.WorldMessageNoticeBody(l)(cmd.Body.Message)
		case "POP_UP":
			bodyProducer = writer.WorldMessagePopUpBody(l)(cmd.Body.Message)
		case "PINK_TEXT":
			bodyProducer = writer.WorldMessagePinkTextBody(l)("", "", cmd.Body.Message)
		case "BLUE_TEXT":
			bodyProducer = writer.WorldMessageBlueTextBody(l)("", "", cmd.Body.Message)
		default:
			l.Warnf("Unknown message type: %s, defaulting to PINK_TEXT", cmd.Body.MessageType)
			bodyProducer = writer.WorldMessagePinkTextBody(l)("", "", cmd.Body.Message)
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.WorldMessage)(bodyProducer))
		if err != nil {
			l.WithError(err).Errorf("Unable to send message to character [%d].", cmd.CharacterId)
		}
	}
}

func handlePlayPortalSound(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.PlayPortalSoundBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.PlayPortalSoundBody]) {
		if cmd.Type != system_message2.CommandPlayPortalSound {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.CharacterEffect)(writer.CharacterPlayPortalSoundEffectEffectBody(l)()))
		if err != nil {
			l.WithError(err).Errorf("Unable to play portal sound for character [%d].", cmd.CharacterId)
		}
	}
}

func handleShowInfo(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.ShowInfoBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.ShowInfoBody]) {
		if cmd.Type != system_message2.CommandShowInfo {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.CharacterEffect)(writer.CharacterShowInfoEffectBody(l)(cmd.Body.Path)))
		if err != nil {
			l.WithError(err).Errorf("Unable to show info for character [%d].", cmd.CharacterId)
		}
	}
}

func handleShowInfoText(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.ShowInfoTextBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.ShowInfoTextBody]) {
		if cmd.Type != system_message2.CommandShowInfoText {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(writer.CharacterStatusMessageOperationSystemMessageBody(l)(cmd.Body.Text)))
		if err != nil {
			l.WithError(err).Errorf("Unable to show info text for character [%d].", cmd.CharacterId)
		}
	}
}

func handleUpdateAreaInfo(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.UpdateAreaInfoBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.UpdateAreaInfoBody]) {
		if cmd.Type != system_message2.CommandUpdateAreaInfo {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(writer.CharacterStatusMessageOperationQuestRecordExBody(l)(cmd.Body.Area, cmd.Body.Info)))
		if err != nil {
			l.WithError(err).Errorf("Unable to update area info for character [%d].", cmd.CharacterId)
		}
	}
}

func handleShowHint(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.ShowHintBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.ShowHintBody]) {
		if cmd.Type != system_message2.CommandShowHint {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.CharacterHint)(writer.CharacterHintBody(cmd.Body.Hint, cmd.Body.Width, cmd.Body.Height, false, 0, 0)))
		if err != nil {
			l.WithError(err).Errorf("Unable to show hint for character [%d].", cmd.CharacterId)
		}
	}
}

func handleShowGuideHint(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.ShowGuideHintBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.ShowGuideHintBody]) {
		if cmd.Type != system_message2.CommandShowGuideHint {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		// Use default duration of 7000ms if not specified
		duration := cmd.Body.Duration
		if duration == 0 {
			duration = 7000
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.GuideTalk)(writer.GuideTalkIdxBody(l)(cmd.Body.HintId, duration)))
		if err != nil {
			l.WithError(err).Errorf("Unable to show guide hint for character [%d].", cmd.CharacterId)
		}
	}
}

func handleShowIntro(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.ShowIntroBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.ShowIntroBody]) {
		if cmd.Type != system_message2.CommandShowIntro {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.CharacterEffect)(writer.CharacterShowIntroEffectBody(l)(cmd.Body.Path)))
		if err != nil {
			l.WithError(err).Errorf("Unable to show intro for character [%d].", cmd.CharacterId)
		}
	}
}

func handleFieldEffect(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.FieldEffectBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.FieldEffectBody]) {
		if cmd.Type != system_message2.CommandFieldEffect {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.FieldEffect)(writer.FieldEffectScreenBody(l)(cmd.Body.Path)))
		if err != nil {
			l.WithError(err).Errorf("Unable to show field effect for character [%d].", cmd.CharacterId)
		}
	}
}

func handleUiLock(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.UiLockBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.UiLockBody]) {
		if cmd.Type != system_message2.CommandUiLock {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.UiLock)(writer.UiLockBody(t)(cmd.Body.Enable, 0)))
		if err != nil {
			l.WithError(err).Errorf("Unable to lock/unlock UI for character [%d].", cmd.CharacterId)
		}
	}
}

func handleUiDisable(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.UiDisableBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.UiDisableBody]) {
		if cmd.Type != system_message2.CommandUiDisable {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.UiDisable)(writer.UiDisableBody(t)(cmd.Body.Enable)))
		if err != nil {
			l.WithError(err).Errorf("Unable to disable/enable UI for character [%d].", cmd.CharacterId)
		}
	}
}
