package character

import (
	"atlas-maps/character/location"
	"atlas-maps/data/map/info"
	consumer2 "atlas-maps/kafka/consumer"
	characterKafka "atlas-maps/kafka/message/character"
	"atlas-maps/kafka/producer"
	_map "atlas-maps/map"
	"atlas-maps/map/timer"
	"atlas-maps/visit"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("status_event")(characterKafka.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("channel_change_request")(characterKafka.EnvCommandTopicChannelChangeRequest)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("character_command")(characterKafka.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(characterKafka.EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLoginFunc(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogoutFunc(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChangedFunc(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChangedFunc(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeletedFunc(l, db)))); err != nil {
			return err
		}
		t, _ = topic.EnvProvider(l)(characterKafka.EnvCommandTopicChannelChangeRequest)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChannelChangeRequestFunc(db)))); err != nil {
			return err
		}
		t, _ = topic.EnvProvider(l)(characterKafka.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeMapFunc(db)))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventLoginFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventLoginBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventLoginBody]) {
		if event.Type == characterKafka.EventCharacterStatusTypeLogin {
			l.Debugf("Character [%d] has logged in. worldId [%d] channelId [%d] mapId [%d] instance [%s].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId, event.Body.Instance)
			transactionId := uuid.New()
			f := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).SetInstance(event.Body.Instance).Build()
			p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), db)
			_ = p.EnterAndEmit(transactionId, f, event.CharacterId)
			if _, err := location.NewProcessor(l, ctx, db).Set(event.CharacterId, f); err != nil {
				l.WithError(err).Warnf("location.Set on LOGIN failed for character [%d].", event.CharacterId)
			}
		}
	}
}

func handleStatusEventLogoutFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]) {
		if event.Type == characterKafka.EventCharacterStatusTypeLogout {
			l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d] instance [%s].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.MapId, event.Body.Instance)
			transactionId := uuid.New()
			current := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).SetInstance(event.Body.Instance).Build()

			lp := location.NewProcessor(l, ctx, db)
			resolved, reason, err := lp.Resolve(current)
			if err != nil {
				l.WithError(err).Warnf("location.Resolve on LOGOUT failed for [%d]; staying put.", event.CharacterId)
				resolved = current
				reason = location.ReasonStayPut
			}
			if reason != location.ReasonStayPut {
				l.WithFields(logrus.Fields{
					"character_id":      event.CharacterId,
					"current_map_id":    current.MapId(),
					"resolved_map_id":   resolved.MapId(),
					"resolution_reason": string(reason),
				}).Info("forced-return resolution on LOGOUT")
			}
			if _, err := lp.Set(event.CharacterId, resolved); err != nil {
				l.WithError(err).Warnf("location.Set on LOGOUT failed for character [%d].", event.CharacterId)
			}

			p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), db)
			_ = p.ExitAndEmit(transactionId, current, event.CharacterId)
		}
	}
}

func handleStatusEventMapChangedFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventMapChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventMapChangedBody]) {
		if event.Type == characterKafka.EventCharacterStatusTypeMapChanged {
			l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] oldInstance [%s] newMapId [%d] newInstance [%s].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.OldMapId, event.Body.OldInstance, event.Body.TargetMapId, event.Body.TargetInstance)
			transactionId := uuid.New()
			newField := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.TargetMapId).SetInstance(event.Body.TargetInstance).Build()
			oldField := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.OldMapId).SetInstance(event.Body.OldInstance).Build()
			p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), db)
			_ = p.TransitionMapAndEmit(transactionId, newField, event.CharacterId, oldField)
			if _, err := location.NewProcessor(l, ctx, db).Set(event.CharacterId, newField); err != nil {
				l.WithError(err).Warnf("location.Set on MAP_CHANGED failed for character [%d].", event.CharacterId)
			}

			// --- map-time-limit timer hooks (task-050) ---
			tp := timer.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
			tp.CancelIfTracked(event.CharacterId)
			md, err := info.NewProcessor(l, ctx).GetById(event.Body.TargetMapId)
			if err != nil {
				l.WithError(err).Debugf("MapTimer: skipping registration for character [%d] target map [%d]; map info unavailable.", event.CharacterId, event.Body.TargetMapId)
			} else if md.IsTimeLimited() {
				if err := tp.Register(transactionId, event.CharacterId, newField, md.ForcedReturnMapId(), uint32(md.TimeLimit())); err != nil {
					l.WithError(err).Warnf("MapTimer: failed to register timer for character [%d] map [%d].", event.CharacterId, event.Body.TargetMapId)
				}
			}
		}
	}
}

func handleStatusEventChannelChangedFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]) {
		if event.Type == characterKafka.EventCharacterStatusTypeChannelChanged {
			l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d] instance [%s].", event.CharacterId, event.WorldId, event.Body.ChannelId, event.Body.OldChannelId, event.Body.Instance)
			transactionId := uuid.New()
			newField := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).SetInstance(event.Body.Instance).Build()
			p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), db)
			_ = p.TransitionChannelAndEmit(transactionId, newField, event.Body.OldChannelId, event.CharacterId)
			if _, err := location.NewProcessor(l, ctx, db).Set(event.CharacterId, newField); err != nil {
				l.WithError(err).Warnf("location.Set on CHANNEL_CHANGED failed for character [%d].", event.CharacterId)
			}

			// --- map-time-limit timer hooks (task-050) ---
			tp := timer.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
			_ = tp.ForceReturnIfTracked(event.CharacterId)
		}
	}
}

func handleStatusEventDeletedFunc(l logrus.FieldLogger, db *gorm.DB) func(logrus.FieldLogger, context.Context, characterKafka.StatusEvent[characterKafka.StatusEventDeletedBody]) {
	return func(fl logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventDeletedBody]) {
		if event.Type == characterKafka.EventCharacterStatusTypeDeleted {
			fl.Debugf("Character [%d] has been deleted.", event.CharacterId)
			if db != nil {
				vp := visit.NewProcessor(fl, ctx, db)
				count, err := vp.DeleteByCharacterId(event.CharacterId)
				if err != nil {
					fl.WithError(err).Errorf("Failed to delete visits for character [%d].", event.CharacterId)
					return
				}
				fl.Debugf("Deleted [%d] visit records for character [%d].", count, event.CharacterId)
			}
		}
	}
}
