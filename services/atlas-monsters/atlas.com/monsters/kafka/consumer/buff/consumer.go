// Package buff consumes character buff-status events (EVENT_TOPIC_CHARACTER_BUFF_STATUS)
// and reacts ONLY to the SuperGmHide (9101004) source: APPLIED relinquishes
// and reassigns the hiding character's controlled monsters (FR-2), EXPIRED
// restores their controller candidacy and re-runs election for uncontrolled
// monsters in their field (FR-3). Every other buff — including Dark Sight —
// passes through untouched.
package buff

import (
	consumer2 "atlas-monsters/kafka/consumer"
	buff2 "atlas-monsters/kafka/message/buff"
	"atlas-monsters/monster"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_buff_status_event")(buff2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(buff2.EnvEventStatusTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventApplied))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExpired))); err != nil {
			return err
		}
		return nil
	}
}

// handleStatusEventApplied reacts ONLY to SuperGmHide (9101004) APPLIED
// events (FR-1.1/FR-1.2). Dark Sight and every other buff pass through
// untouched. GmHideId (9001004) is absent from v83 game data and is
// deliberately not handled.
func handleStatusEventApplied(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
	if e.Type != buff2.EventStatusTypeBuffApplied {
		return
	}
	if e.Body.SourceId != int32(skill.SuperGmHideId) {
		return
	}
	if err := monster.NewProcessor(l, ctx).RelinquishControlOnHide(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to relinquish monster control for hiding character [%d].", e.CharacterId)
	}
}

// handleStatusEventExpired reacts ONLY to SuperGmHide (9101004) EXPIRED
// events. Dark Sight and every other buff pass through untouched.
func handleStatusEventExpired(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
	if e.Type != buff2.EventStatusTypeBuffExpired {
		return
	}
	if e.Body.SourceId != int32(skill.SuperGmHideId) {
		return
	}
	if err := monster.NewProcessor(l, ctx).RestoreCandidacyOnReveal(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to restore controller candidacy for revealed character [%d].", e.CharacterId)
	}
}
