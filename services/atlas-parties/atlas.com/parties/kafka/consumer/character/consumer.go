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
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, event StatusEvent[StatusEventLoginBody]) {
	if event.Type != StatusEventTypeLogin {
		return
	}
	err := character.Login(l)(ctx)(byte(event.WorldId), byte(event.Body.ChannelId), uint32(event.Body.MapId), event.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process login for character [%d].", event.CharacterId)
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event StatusEvent[StatusEventLogoutBody]) {
	if event.Type != StatusEventTypeLogout {
		return
	}
	err := character.Logout(l)(ctx)(event.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process logout for character [%d].", event.CharacterId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[ChangeChannelEventLoginBody]) {
	if e.Type != StatusEventTypeChannelChanged {
		return
	}
	err := character.ChannelChange(l)(ctx)(e.CharacterId, byte(e.Body.ChannelId))
	if err != nil {
		l.WithError(err).Errorf("Unable to process channel changed for character [%d].", e.CharacterId)
	}
}

func handleMapChangedStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event StatusEvent[StatusEventMapChangedBody]) {
	if event.Type != StatusEventTypeMapChanged {
		return
	}
	err := character.MapChange(l)(ctx)(event.CharacterId, uint32(event.Body.TargetMapId))
	if err != nil {
		l.WithError(err).Errorf("Unable to process map changed for character [%d].", event.CharacterId)
	}
}

func handleStatusEventDeleted(l logrus.FieldLogger, ctx context.Context, event StatusEvent[StatusEventDeletedBody]) {
	if event.Type != StatusEventTypeDeleted {
		return
	}
	
	l.WithField("transactionId", event.TransactionId).
		WithField("worldId", event.WorldId).
		WithField("characterId", event.CharacterId).
		Debugf("Processing character deletion event for character [%d].", event.CharacterId)
	
	// First, validate if character is in a party and log the party information
	p, err := party.GetByCharacter(ctx)(event.CharacterId)
	if err == nil {
		l.WithField("transactionId", event.TransactionId).
			WithField("partyId", p.Id()).
			WithField("isLeader", party.IsLeader(p, event.CharacterId)).
			WithField("partyMemberCount", len(p.Members())).
			Debugf("Character [%d] found in party [%d] before deletion.", event.CharacterId, p.Id())
		
		// Validate party integrity before deletion
		if err := party.ValidatePartyIntegrity(p); err != nil {
			l.WithError(err).
				WithField("transactionId", event.TransactionId).
				WithField("partyId", p.Id()).
				Warnf("Party [%d] integrity validation failed before character deletion.", p.Id())
		}
	} else {
		l.WithField("transactionId", event.TransactionId).
			Debugf("Character [%d] not found in any party before deletion.", event.CharacterId)
	}
	
	// Remove character from party using the new removal logic
	removedParty, err := party.RemoveCharacterFromParty(l)(ctx)(event.CharacterId)
	if err != nil {
		l.WithError(err).
			WithField("transactionId", event.TransactionId).
			WithField("worldId", event.WorldId).
			WithField("characterId", event.CharacterId).
			Errorf("Unable to remove character [%d] from party during deletion.", event.CharacterId)
	} else {
		if removedParty.Id() != 0 {
			l.WithField("transactionId", event.TransactionId).
				WithField("partyId", removedParty.Id()).
				WithField("remainingMembers", len(removedParty.Members())).
				WithField("newLeader", removedParty.LeaderId()).
				Debugf("Character [%d] successfully removed from party [%d].", event.CharacterId, removedParty.Id())
		}
	}
	
	// Also process character deletion from character registry
	err = character.Delete(l)(ctx)(event.CharacterId)
	if err != nil {
		l.WithError(err).
			WithField("transactionId", event.TransactionId).
			WithField("worldId", event.WorldId).
			WithField("characterId", event.CharacterId).
			Errorf("Unable to process character deletion for character [%d].", event.CharacterId)
	} else {
		// Log cache statistics for monitoring
		hits, misses, hitRate := party.GetCacheStats(ctx)
		l.WithField("transactionId", event.TransactionId).
			WithField("worldId", event.WorldId).
			WithField("characterId", event.CharacterId).
			WithField("cacheHits", hits).
			WithField("cacheMisses", misses).
			WithField("cacheHitRate", hitRate).
			WithField("cacheSize", party.GetCacheSize(ctx)).
			Infof("Successfully processed character deletion for character [%d].", event.CharacterId)
	}
}
