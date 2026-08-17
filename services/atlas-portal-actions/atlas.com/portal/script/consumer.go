package script

import (
	"atlas-portal-actions/character"
	"atlas-portal-actions/dedupe"
	"context"

	consumer2 "atlas-portal-actions/kafka/consumer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// commandEvent represents a Kafka command message
type commandEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	PortalId  uint32     `json:"portalId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// enterBody represents the body of an enter portal command
type enterBody struct {
	CharacterId uint32 `json:"characterId"`
	PortalName  string `json:"portalName"`
}

// InitConsumers initializes Kafka consumers for portal actions
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
		return func(groupId string) {
			rf(
				consumer2.NewConfig(l)("portal_actions_command")(EnvCommandTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser),
			)
		}
	}
}

// InitHandlers initializes Kafka message handlers
func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterCommandFunc(l, db)))); err != nil {
			return err
		}
		return nil
	}
}

// handleEnterCommandFunc returns a handler function for enter commands
func handleEnterCommandFunc(l logrus.FieldLogger, db *gorm.DB) func(logrus.FieldLogger, context.Context, commandEvent[enterBody]) {
	return func(fl logrus.FieldLogger, ctx context.Context, command commandEvent[enterBody]) {
		handleEnterCommand(l, ctx, db, command)
	}
}

// Package seams. Production wiring is unchanged; tests substitute these to
// observe handleEnterCommand's unlock decision without Kafka or a database.
var (
	newScriptProcessorFn = func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
		return NewProcessor(l, ctx, db)
	}
	enableActionsFn = character.EnableActions
	gateFn          = dedupe.GetGate
)

// handleEnterCommand handles a portal enter command
func handleEnterCommand(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, c commandEvent[enterBody]) {
	l.Debugf("Received portal enter command for character [%d] on portal [%s] (id=%d) in map [%d]",
		c.Body.CharacterId, c.Body.PortalName, c.PortalId, c.MapId)

	// Create field model from command
	ch := channel.NewModel(c.WorldId, c.ChannelId)
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()

	// Duplicate-command gate (task-184 FR-3.1). Evaluated before any script
	// load, condition evaluation, or operation dispatch, so a duplicate has no
	// side effect at all. Fails open on a Redis error — FR-2's conditional
	// unlock is the primary fix and stands on its own; this must never become a
	// single point of failure for portal traversal.
	//
	// A non-zero drop rate here means some outcome path is still unlocking a
	// character whose warp is in flight.
	if !gateFn().Allow(l, ctx, dedupe.Key{
		CharacterId: c.Body.CharacterId,
		MapId:       c.MapId,
		Instance:    c.Instance,
		PortalId:    c.PortalId,
	}) {
		return
	}

	// Create processor with tenant context from Kafka message
	processor := newScriptProcessorFn(l, ctx, db)

	// Process the portal script (pass numeric portalId for use in operations like block_portal)
	result := processor.Process(f, c.Body.CharacterId, c.Body.PortalName, c.PortalId)

	if result.Error != nil {
		l.WithError(result.Error).Errorf("Failed to process portal script [%s] for character [%d]",
			c.Body.PortalName, c.Body.CharacterId)
	} else {
		l.Debugf("Portal script [%s] result: allow=%t, matchedRule=%s, characterMoved=%t",
			c.Body.PortalName, result.Allow, result.MatchedRule, result.CharacterMoved)
	}

	// An outcome that dispatched a warp is unlocked by the SET_FIELD that warp
	// produces — CWvsContext::OnGameStageChanged clears m_bExclRequestSent on
	// every set_stage. Clearing it HERE, while the warp is still in flight and
	// the player still overlaps the portal's collision rect, is what makes the
	// GMS v83 client legitimately re-fire the ENTER request and execute the
	// whole rule a second time. See
	// docs/tasks/task-184-portal-enter-double-execute/prd.md §1.1.
	//
	// If the warp never lands, the saga's 5s timeout fails it and
	// kafka/consumer/saga/consumer.go handleStatusEventFailed unlocks the
	// player from the PendingAction registered in executor.go.
	//
	// CharacterMoved means "successfully dispatched", not "declared": a warp
	// that failed before creating its saga leaves this false, so the player is
	// unlocked here rather than waiting on a saga that does not exist.
	if result.CharacterMoved {
		return
	}
	enableActionsFn(l)(ctx)(ch, c.Body.CharacterId)
}
