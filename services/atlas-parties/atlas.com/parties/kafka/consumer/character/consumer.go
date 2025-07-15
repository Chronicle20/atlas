package character

import (
	"atlas-parties/character"
	consumer2 "atlas-parties/kafka/consumer"
	"atlas-parties/party"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMapChangedStatusEventLogout)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLevelChanged)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventJobChanged)))
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusEventLoginBody]) {
	if e.Type != StatusEventTypeLogin {
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("channelId", e.Body.ChannelId).
		WithField("mapId", e.Body.MapId).
		Debugf("Processing login event for character [%d].", e.CharacterId)

	err := character.NewProcessor(l, ctx).LoginAndEmit(byte(e.WorldId), byte(e.Body.ChannelId), uint32(e.Body.MapId), e.CharacterId)
	if err != nil {
		l.WithError(err).
			WithField("characterId", e.CharacterId).
			WithField("worldId", e.WorldId).
			WithField("transactionId", e.TransactionId).
			WithField("channelId", e.Body.ChannelId).
			WithField("mapId", e.Body.MapId).
			Errorf("Unable to process login for character [%d].", e.CharacterId)
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		Debugf("Successfully processed login for character [%d].", e.CharacterId)

}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusEventLogoutBody]) {
	if e.Type != StatusEventTypeLogout {
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("channelId", e.Body.ChannelId).
		WithField("mapId", e.Body.MapId).
		Debugf("Processing logout event for character [%d].", e.CharacterId)

	err := character.NewProcessor(l, ctx).LogoutAndEmit(e.CharacterId)
	if err != nil {
		l.WithError(err).
			WithField("characterId", e.CharacterId).
			WithField("worldId", e.WorldId).
			WithField("transactionId", e.TransactionId).
			WithField("channelId", e.Body.ChannelId).
			WithField("mapId", e.Body.MapId).
			Errorf("Unable to process logout for character [%d].", e.CharacterId)
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		Debugf("Successfully processed logout for character [%d].", e.CharacterId)

}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[ChangeChannelEventLoginBody]) {
	if e.Type != StatusEventTypeChannelChanged {
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("newChannelId", e.Body.ChannelId).
		WithField("oldChannelId", e.Body.OldChannelId).
		WithField("mapId", e.Body.MapId).
		Debugf("Processing channel change event for character [%d].", e.CharacterId)

	err := character.NewProcessor(l, ctx).ChannelChange(e.CharacterId, byte(e.Body.ChannelId))
	if err != nil {
		l.WithError(err).
			WithField("characterId", e.CharacterId).
			WithField("worldId", e.WorldId).
			WithField("transactionId", e.TransactionId).
			WithField("newChannelId", e.Body.ChannelId).
			WithField("oldChannelId", e.Body.OldChannelId).
			Errorf("Unable to process channel changed for character [%d].", e.CharacterId)
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("newChannelId", e.Body.ChannelId).
		Debugf("Successfully processed channel change for character [%d].", e.CharacterId)
}

func handleMapChangedStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusEventMapChangedBody]) {
	if e.Type != StatusEventTypeMapChanged {
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("oldMapId", e.Body.OldMapId).
		WithField("targetMapId", e.Body.TargetMapId).
		WithField("targetPortalId", e.Body.TargetPortalId).
		WithField("channelId", e.Body.ChannelId).
		Debugf("Processing map change event for character [%d].", e.CharacterId)

	err := character.NewProcessor(l, ctx).MapChange(e.CharacterId, uint32(e.Body.TargetMapId))
	if err != nil {
		l.WithError(err).
			WithField("characterId", e.CharacterId).
			WithField("worldId", e.WorldId).
			WithField("transactionId", e.TransactionId).
			WithField("oldMapId", e.Body.OldMapId).
			WithField("targetMapId", e.Body.TargetMapId).
			WithField("targetPortalId", e.Body.TargetPortalId).
			Errorf("Unable to process map changed for character [%d].", e.CharacterId)
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("targetMapId", e.Body.TargetMapId).
		Debugf("Successfully processed map change for character [%d].", e.CharacterId)
}

func handleStatusEventDeleted(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusEventDeletedBody]) {
	// Early validation: check event type
	if e.Type != StatusEventTypeDeleted {
		return
	}

	l.WithField("transactionId", e.TransactionId).
		WithField("worldId", e.WorldId).
		WithField("characterId", e.CharacterId).
		Debugf("Processing character deletion event for character [%d].", e.CharacterId)

	// First, validate if character is in a party and log the party information
	pp := party.NewProcessor(l, ctx)
	p, err := pp.GetByCharacter(e.CharacterId)
	if err != nil {
		l.WithField("transactionId", e.TransactionId).
			Debugf("Character [%d] not found in any party before deletion.", e.CharacterId)
	}

	l.WithField("transactionId", e.TransactionId).
		WithField("partyId", p.Id()).
		WithField("isLeader", p.LeaderId() == e.CharacterId).
		WithField("partyMemberCount", len(p.Members())).
		Debugf("Character [%d] found in party [%d] before deletion.", e.CharacterId, p.Id())

	p, err = pp.LeaveAndEmit(p.Id(), e.CharacterId)
	if err != nil {
		l.WithError(err).
			WithField("transactionId", e.TransactionId).
			WithField("worldId", e.WorldId).
			WithField("characterId", e.CharacterId).
			Errorf("Unable to remove character [%d] from party during deletion.", e.CharacterId)
	} else {
		if p.Id() != 0 {
			l.WithField("transactionId", e.TransactionId).
				WithField("partyId", p.Id()).
				WithField("remainingMembers", len(p.Members())).
				WithField("newLeader", p.LeaderId()).
				Debugf("Character [%d] successfully removed from party [%d].", e.CharacterId, p.Id())
		}
	}

	err = character.NewProcessor(l, ctx).Delete(e.CharacterId)
	if err != nil {
		l.WithError(err).
			WithField("transactionId", e.TransactionId).
			WithField("worldId", e.WorldId).
			WithField("characterId", e.CharacterId).
			Errorf("Unable to process character deletion for character [%d].", e.CharacterId)
	} else {
		// Log cache statistics for monitoring
		l.WithField("transactionId", e.TransactionId).
			WithField("worldId", e.WorldId).
			WithField("characterId", e.CharacterId).
			Infof("Successfully processed character deletion for character [%d].", e.CharacterId)
	}
}

func handleStatusEventLevelChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[LevelChangedStatusEventBody]) {
	if e.Type != StatusEventTypeLevelChanged {
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("channelId", e.Body.ChannelId).
		WithField("currentLevel", e.Body.Current).
		WithField("levelAmount", e.Body.Amount).
		Debugf("Processing level changed event for character [%d].", e.CharacterId)

	err := character.NewProcessor(l, ctx).LevelChangeAndEmit(e.CharacterId, e.Body.Current)
	if err != nil {
		l.WithError(err).
			WithField("characterId", e.CharacterId).
			WithField("worldId", e.WorldId).
			WithField("transactionId", e.TransactionId).
			WithField("channelId", e.Body.ChannelId).
			WithField("currentLevel", e.Body.Current).
			Errorf("Unable to process level change for character [%d].", e.CharacterId)
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("currentLevel", e.Body.Current).
		Debugf("Successfully processed level change for character [%d].", e.CharacterId)
}

func handleStatusEventJobChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[JobChangedStatusEventBody]) {
	if e.Type != StatusEventTypeJobChanged {
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("channelId", e.Body.ChannelId).
		WithField("jobId", e.Body.JobId).
		Debugf("Processing job changed event for character [%d].", e.CharacterId)

	err := character.NewProcessor(l, ctx).JobChangeAndEmit(e.CharacterId, e.Body.JobId)
	if err != nil {
		l.WithError(err).
			WithField("characterId", e.CharacterId).
			WithField("worldId", e.WorldId).
			WithField("transactionId", e.TransactionId).
			WithField("channelId", e.Body.ChannelId).
			WithField("jobId", e.Body.JobId).
			Errorf("Unable to process job change for character [%d].", e.CharacterId)
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("jobId", e.Body.JobId).
		Debugf("Successfully processed job change for character [%d].", e.CharacterId)
}
