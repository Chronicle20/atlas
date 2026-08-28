package skill

import (
	skillmodel "atlas-channel/character/skill"
	"atlas-channel/character/snapshot"
	consumer2 "atlas-channel/kafka/consumer"
	skill2 "atlas-channel/kafka/message/skill"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("skill_status_event")(skill2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(skill2.EnvStatusEventTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleCooldownApplied(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleCooldownExpired(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSnapshotSkillCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSnapshotSkillUpdated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSnapshotSkillDeleted(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleCreated(sc server.Model, wp writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventCreatedBody]) {
		if e.Type != skill2.StatusEventTypeCreated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, announceSkillUpdate(l)(ctx)(wp)(e.SkillId, e.Body.Level, e.Body.MasterLevel, e.Body.Expiration))
		if err != nil {
			l.WithError(err).Errorf("Unable to update character [%d] skill [%d].", e.CharacterId, e.SkillId)
		}
	}
}

func handleUpdated(sc server.Model, wp writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventUpdatedBody]) {
		if e.Type != skill2.StatusEventTypeUpdated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, announceSkillUpdate(l)(ctx)(wp)(e.SkillId, e.Body.Level, e.Body.MasterLevel, e.Body.Expiration))
		if err != nil {
			l.WithError(err).Errorf("Unable to update character [%d] skill [%d].", e.CharacterId, e.SkillId)
		}
	}
}

func announceSkillUpdate(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, level byte, masterLevel byte, expiration time.Time) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, level byte, masterLevel byte, expiration time.Time) model.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32, level byte, masterLevel byte, expiration time.Time) model.Operator[session.Model] {
			return func(skillId uint32, level byte, masterLevel byte, expiration time.Time) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charpkt.CharacterSkillChangeWriter)(charpkt.NewCharacterSkillChange(true, skillId, level, masterLevel, expiration, true).Encode)
			}
		}
	}
}

func handleCooldownApplied(sc server.Model, wp writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventCooldownAppliedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventCooldownAppliedBody]) {
		if e.Type != skill2.StatusEventTypeCooldownApplied {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, announceSkillCooldown(l)(ctx)(wp)(e.SkillId, e.Body.CooldownExpiresAt))
		if err != nil {
			l.WithError(err).Errorf("Unable to update character [%d] skill [%d].", e.CharacterId, e.SkillId)
		}
	}
}

func announceSkillCooldown(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, cooldownExpiresAt time.Time) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, cooldownExpiresAt time.Time) model.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32, cooldownExpiresAt time.Time) model.Operator[session.Model] {
			return func(skillId uint32, cooldownExpiresAt time.Time) model.Operator[session.Model] {
				var cd uint16
				if !cooldownExpiresAt.IsZero() {
					cd = uint16(cooldownExpiresAt.Sub(time.Now()).Seconds())
				}
				return session.Announce(l)(ctx)(wp)(charpkt.CharacterSkillCooldownWriter)(charpkt.NewCharacterSkillCooldown(skillId, cd).Encode)
			}
		}
	}
}

func handleCooldownExpired(sc server.Model, wp writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventCooldownExpiredBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventCooldownExpiredBody]) {
		if e.Type != skill2.StatusEventTypeCooldownExpired {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, announceSkillCooldownReset(l)(ctx)(wp)(e.SkillId))
		if err != nil {
			l.WithError(err).Errorf("Unable to update character [%d] skill [%d].", e.CharacterId, e.SkillId)
		}
	}
}

func announceSkillCooldownReset(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32) model.Operator[session.Model] {
			return func(skillId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charpkt.CharacterSkillCooldownWriter)(charpkt.NewCharacterSkillCooldown(skillId, 0).Encode)
			}
		}
	}
}

// --- task-122 snapshot maintenance (additive) ---

func snapshotSkillFromEvent(skillId uint32, level byte, masterLevel byte, expiration time.Time) skillmodel.Model {
	return skillmodel.NewModelBuilder(skillconst.Id(skillId)).
		SetLevel(level).
		SetMasterLevel(masterLevel).
		SetExpiration(expiration).
		MustBuild()
}

func handleSnapshotSkillCreated(sc server.Model, _ writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventCreatedBody]) {
		if e.Type != skill2.StatusEventTypeCreated {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().UpsertSkill(t, e.CharacterId, snapshotSkillFromEvent(e.SkillId, e.Body.Level, e.Body.MasterLevel, e.Body.Expiration))
	}
}

func handleSnapshotSkillUpdated(sc server.Model, _ writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventUpdatedBody]) {
		if e.Type != skill2.StatusEventTypeUpdated {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().UpsertSkill(t, e.CharacterId, snapshotSkillFromEvent(e.SkillId, e.Body.Level, e.Body.MasterLevel, e.Body.Expiration))
	}
}

// handleSnapshotSkillDeleted is atlas-channel's first consumer of the
// skill DELETED event (saga compensation path; the packet layer never
// needed it — event-coverage.md §3).
func handleSnapshotSkillDeleted(sc server.Model, _ writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventDeletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventDeletedBody]) {
		if e.Type != skill2.StatusEventTypeDeleted {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().RemoveSkill(t, e.CharacterId, skillconst.Id(e.SkillId))
	}
}
