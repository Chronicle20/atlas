// Package invite consumes EVENT_TOPIC_INVITE_STATUS, the answer half of the
// trade invite atlas-trades issues on COMMAND_TOPIC_INVITE. The topic carries
// every invite family, so each handler filters on BOTH the status type and
// InviteType == invite.TypeTrade before it touches a room.
//
// FR-2.6 (invite expiry) needs no third arm: atlas-invites' timeout task emits a
// REJECTED status for an expired invite
// (services/atlas-invites/atlas.com/invites/invite/task.go:43-54), so an expiry
// arrives on the same path as a decline.
package invite

import (
	consumer2 "atlas-trades/kafka/consumer"
	invitemsg "atlas-trades/kafka/message/invite"
	"atlas-trades/trade"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/invite"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("trade_invite_status")(invitemsg.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			var err error
			t, err = topic.EnvProvider(l)(invitemsg.EnvEventStatusTopic)()
			if err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAccepted(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRejected(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleAccepted seats the acceptor. The field comes from the ROOM rather than
// the event: EVENT_TOPIC_INVITE_STATUS carries only a world id, and the room
// already knows the channel, map and instance the trade was opened in.
func handleAccepted(db *gorm.DB) message.Handler[invitemsg.StatusEvent[invitemsg.AcceptedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e invitemsg.StatusEvent[invitemsg.AcceptedEventBody]) {
		if e.Type != invite.StatusTypeAccepted || e.InviteType != invite.TypeTrade {
			return
		}
		p := trade.NewProcessor(l, ctx, db)
		room, ok := p.RoomByHandle(uint32(e.ReferenceId))
		if !ok {
			l.Errorf("Character [%d] accepted trade invite [%d], but its room is gone.", e.Body.TargetId, e.ReferenceId)
			return
		}
		if err := p.EnterRoom(e.TransactionId, room.Field(), e.Body.TargetId, uint32(e.ReferenceId), room.RoomType()); err != nil {
			l.WithError(err).Errorf("Unable to seat character [%d] in trade room [%d].", e.Body.TargetId, e.ReferenceId)
		}
	}
}

// handleRejected covers both an explicit decline and an expired invite. It uses
// InviteRejected rather than DeclineInvite because atlas-invites has already
// retired the offer by the time it emits this — sending a reject back would only
// make it fail to find the invite it just deleted.
func handleRejected(db *gorm.DB) message.Handler[invitemsg.StatusEvent[invitemsg.RejectedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e invitemsg.StatusEvent[invitemsg.RejectedEventBody]) {
		if e.Type != invite.StatusTypeRejected || e.InviteType != invite.TypeTrade {
			return
		}
		if err := trade.NewProcessor(l, ctx, db).InviteRejected(e.TransactionId, e.Body.TargetId, e.Body.OriginatorId); err != nil {
			l.WithError(err).Errorf("Unable to tear down character [%d]'s trade room after [%d] declined.", e.Body.OriginatorId, e.Body.TargetId)
		}
	}
}
