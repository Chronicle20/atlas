// Package buff consumes character buff-status events (EVENT_TOPIC_CHARACTER_BUFF_STATUS)
// and reacts ONLY to the SuperGmHide source (wire 9101004 at v0.62+, wire
// 5101004 at v0.48 -- resolved via the tenant's version set, task-187):
// APPLIED relinquishes and reassigns the hiding character's controlled
// monsters (FR-2), EXPIRED restores their controller candidacy and re-runs
// election for uncontrolled monsters in their field (FR-3). Every other
// buff — including Dark Sight — passes through untouched.
package buff

import (
	consumer2 "atlas-monsters/kafka/consumer"
	buff2 "atlas-monsters/kafka/message/buff"
	"atlas-monsters/monster"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// isSuperGmHideSource reports whether sourceId -- a buff's version-specific
// WIRE skill id (5101004 at v0.48, 9101004 at v0.62+) -- resolves to the
// SuperGmHide identity under t's version set (task-187). A raw compare
// against the canonical wire constant would silently never match a v0.48
// hide buff.
func isSuperGmHideSource(t tenant.Model, sourceId int32) bool {
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skill.Id(sourceId))
	return ok && id == skill.SuperGmHide
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_buff_status_event")(buff2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
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

// handleStatusEventApplied reacts ONLY to SuperGmHide APPLIED events
// (FR-1.1/FR-1.2). Dark Sight and every other buff pass through untouched.
// GmHideId (9001004) is absent from v83 game data and is deliberately not
// handled.
func handleStatusEventApplied(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
	if e.Type != buff2.EventStatusTypeBuffApplied {
		return
	}
	t := tenant.MustFromContext(ctx)
	if !isSuperGmHideSource(t, e.Body.SourceId) {
		return
	}
	if err := monster.NewProcessor(l, ctx).RelinquishControlOnHide(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to relinquish monster control for hiding character [%d].", e.CharacterId)
	}
}

// handleStatusEventExpired reacts ONLY to SuperGmHide EXPIRED events. Dark
// Sight and every other buff pass through untouched.
func handleStatusEventExpired(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
	if e.Type != buff2.EventStatusTypeBuffExpired {
		return
	}
	t := tenant.MustFromContext(ctx)
	if !isSuperGmHideSource(t, e.Body.SourceId) {
		return
	}
	if err := monster.NewProcessor(l, ctx).RestoreCandidacyOnReveal(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to restore controller candidacy for revealed character [%d].", e.CharacterId)
	}
}
