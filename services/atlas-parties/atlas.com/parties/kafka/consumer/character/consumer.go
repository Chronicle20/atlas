package character

import (
	"atlas-parties/character"
	consumer2 "atlas-parties/kafka/consumer"
	"atlas-parties/party"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
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
	// Early validation: check event type
	if event.Type != StatusEventTypeDeleted {
		return
	}
	
	// Comprehensive validation for malformed events
	if err := validateStatusEventDeleted(l, event); err != nil {
		l.WithError(err).
			WithField("eventType", event.Type).
			WithField("characterId", event.CharacterId).
			WithField("worldId", event.WorldId).
			WithField("transactionId", event.TransactionId).
			Errorf("Malformed character deletion event received, skipping processing.")
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
	
	// Remove character from party using the new removal logic with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				l.WithField("transactionId", event.TransactionId).
					WithField("worldId", event.WorldId).
					WithField("characterId", event.CharacterId).
					WithField("panic", r).
					Errorf("Panic occurred during character [%d] removal from party, recovered gracefully.", event.CharacterId)
			}
		}()
		
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
	}()
	
	// Also process character deletion from character registry with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				l.WithField("transactionId", event.TransactionId).
					WithField("worldId", event.WorldId).
					WithField("characterId", event.CharacterId).
					WithField("panic", r).
					Errorf("Panic occurred during character [%d] deletion from registry, recovered gracefully.", event.CharacterId)
			}
		}()
		
		err := character.Delete(l)(ctx)(event.CharacterId)
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
	}()
}

// validateStatusEventDeleted performs comprehensive validation of character deletion events
func validateStatusEventDeleted(l logrus.FieldLogger, event StatusEvent[StatusEventDeletedBody]) error {
	var validationErrors []string
	
	// Validate character ID - must be positive and within reasonable bounds
	if event.CharacterId == 0 {
		validationErrors = append(validationErrors, "invalid character ID: cannot be zero")
	} else if event.CharacterId > 4294967295 { // max uint32
		validationErrors = append(validationErrors, "invalid character ID: exceeds maximum value")
	}
	
	// Validate transaction ID - must not be empty for audit trails
	if event.TransactionId == uuid.Nil {
		l.Warnf("Character deletion event for character [%d] missing transaction ID, processing anyway.", event.CharacterId)
		// Don't add to validationErrors as this is not critical, just log warning
	}
	
	// Validate world ID - must be positive and within reasonable bounds
	if event.WorldId == 0 {
		validationErrors = append(validationErrors, "invalid world ID: cannot be zero")
	} else if event.WorldId > 255 { // reasonable world ID limit for byte type
		validationErrors = append(validationErrors, "invalid world ID: exceeds maximum value")
	}
	
	// Validate event type consistency (double-check)
	if event.Type != StatusEventTypeDeleted {
		validationErrors = append(validationErrors, "event type mismatch: expected DELETED type")
	}
	
	// Check for empty event type string
	if event.Type == "" {
		validationErrors = append(validationErrors, "event type cannot be empty")
	}
	
	// If we have validation errors, combine them and return
	if len(validationErrors) > 0 {
		l.WithField("validationErrors", validationErrors).
			WithField("characterId", event.CharacterId).
			WithField("worldId", event.WorldId).
			WithField("transactionId", event.TransactionId).
			Warnf("Character deletion event failed validation with %d errors.", len(validationErrors))
		
		return errors.New("event validation failed: " + validationErrors[0]) // Return first error
	}
	
	// All validations passed
	return nil
}
