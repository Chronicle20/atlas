package buff

import (
	consumer2 "atlas-mounts/kafka/consumer"
	mountmessage "atlas-mounts/kafka/message"
	buffmsg "atlas-mounts/kafka/message/buff"
	"atlas-mounts/mount"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// Function seams. Production wires these to the real registry + processor; tests
// override them with fakes to exercise the branching logic without Kafka, Redis,
// or a database.
var (
	registryAdd = func(ctx context.Context, characterId uint32, c mount.MountRideContext) error {
		return mount.GetRegistry().Add(ctx, characterId, c)
	}
	registryRemove = func(ctx context.Context, characterId uint32) error {
		return mount.GetRegistry().Remove(ctx, characterId)
	}
	emitSet = func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, worldId world.Id, characterId uint32) error {
		return database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
			p := mount.NewProcessor(l, ctx, tx)
			return mountmessage.Emit(outbox.EmitProvider(l, ctx, tx))(func(mb *mountmessage.Buffer) error {
				return p.EmitSet(mb)(worldId, characterId)
			})
		})
	}
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_buff_status_event")(buffmsg.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(buffmsg.EnvEventStatusTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBuffApplied(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBuffExpired(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// monsterRidingChange returns the MONSTER_RIDING StatChange (if present) from a
// set of buff changes. Its Amount carries the vehicle item id (Task 7 / Task 2
// encoding). The bool reports whether such a change was found.
func monsterRidingChange(changes []buffmsg.StatChange) (buffmsg.StatChange, bool) {
	for _, c := range changes {
		if c.Type == string(characterconst.TemporaryStatTypeMonsterRiding) {
			return c, true
		}
	}
	return buffmsg.StatChange{}, false
}

func handleBuffApplied(db *gorm.DB) message.Handler[buffmsg.StatusEvent[buffmsg.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buffmsg.StatusEvent[buffmsg.AppliedStatusEventBody]) {
		if e.Type != buffmsg.EventStatusTypeBuffApplied {
			return
		}

		mr, ok := monsterRidingChange(e.Body.Changes)
		if !ok {
			// Not a mount buff; ignore.
			return
		}

		skillId := e.Body.SourceId
		vehicleId := mr.Amount

		if skill.IsTamedMountSkill(skill.Id(skillId)) {
			l.Debugf("Tamed mount activated for character [%d]: skill [%d] vehicle [%d].", e.CharacterId, skillId, vehicleId)
			if err := registryAdd(ctx, e.CharacterId, mount.MountRideContext{
				WorldId:   e.WorldId,
				SkillId:   skillId,
				VehicleId: vehicleId,
			}); err != nil {
				l.WithError(err).Errorf("Unable to register active mount for character [%d]. Tiredness will not tick.", e.CharacterId)
				return
			}
		} else {
			// Skill-only mount: no tiredness, so no registry entry (FR-2.2).
			l.Debugf("Skill-only mount activated for character [%d]: skill [%d] vehicle [%d].", e.CharacterId, skillId, vehicleId)
		}

		if err := emitSet(l, ctx, db, e.WorldId, e.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to emit mount SET for character [%d].", e.CharacterId)
		}
	}
}

func handleBuffExpired(db *gorm.DB) message.Handler[buffmsg.StatusEvent[buffmsg.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buffmsg.StatusEvent[buffmsg.ExpiredStatusEventBody]) {
		if e.Type != buffmsg.EventStatusTypeBuffExpired {
			return
		}

		if _, ok := monsterRidingChange(e.Body.Changes); !ok {
			// Not a mount buff; ignore.
			return
		}

		l.Debugf("Mount deactivated for character [%d]: skill [%d].", e.CharacterId, e.Body.SourceId)
		// State is already persisted; just drop the active registry entry so the
		// ticker stops ticking it. Remove on a missing key is a safe no-op, which
		// covers skill-only mounts that were never registered (FR-4.4).
		if err := registryRemove(ctx, e.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to remove active mount for character [%d].", e.CharacterId)
		}
	}
}
