package reactor

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_REACTOR"
)

const (
	CommandTypeHit = "HIT"
)

// Processor is the interface for reactor operations from the saga-orchestrator
type Processor interface {
	// HitReactorByName resolves a reactor by name in a field, then produces a HIT command
	HitReactorByName(f field.Model, characterId uint32, reactorName string) error
	// ResetReactors resets every reactor on the field to state 0 via
	// atlas-reactors' POST .../reactors/reset. minState is optional: nil
	// resets every reactor; non-nil resets only reactors whose state is at
	// least *minState (task-290 G5).
	ResetReactors(f field.Model, minState *int8) error
	// ShuffleReactors randomly permutes the positions of every reactor on
	// the field via atlas-reactors' POST .../reactors/shuffle (task-290 G5).
	ShuffleReactors(f field.Model) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new reactor processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// HitReactorByName resolves a reactor by name via atlas-reactors REST API,
// then produces a HIT command to COMMAND_TOPIC_REACTOR.
func (p *ProcessorImpl) HitReactorByName(f field.Model, characterId uint32, reactorName string) error {
	// Query atlas-reactors for reactor by name in the field
	reactors, err := p.getReactorsByName(f.WorldId(), f.ChannelId(), f.MapId(), f.Instance(), reactorName)
	if err != nil {
		return fmt.Errorf("failed to resolve reactor name [%s]: %w", reactorName, err)
	}

	if len(reactors) == 0 {
		return fmt.Errorf("no reactor found with name [%s] in field (world=%d, channel=%d, map=%d, instance=%s)",
			reactorName, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
	}

	// Hit the first matching reactor
	reactor := reactors[0]
	p.l.WithFields(logrus.Fields{
		"reactor_id":   reactor.Id,
		"reactor_name": reactorName,
	}).Debug("Resolved reactor by name, producing HIT command")

	return p.produceHitCommand(f, reactor.Id, characterId)
}

// getReactorsByName fetches reactors by name from atlas-reactors
func (p *ProcessorImpl) getReactorsByName(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, name string) ([]ReactorRestModel, error) {
	return requests.SliceProvider[ReactorRestModel, ReactorRestModel](p.l, p.ctx)(
		requestReactorsByName(p.ctx, worldId, channelId, mapId, instance, name),
		ExtractReactor,
		model.Filters[ReactorRestModel](),
	)()
}

// ResetReactors resets every reactor on the field to state 0 (optionally
// filtered by a minimum state) via atlas-reactors' POST .../reactors/reset.
func (p *ProcessorImpl) ResetReactors(f field.Model, minState *int8) error {
	_, err := requestResetReactors(p.ctx, f, minState)(p.l, p.ctx)
	if err != nil {
		return fmt.Errorf("failed to reset reactors: %w", err)
	}
	return nil
}

// ShuffleReactors randomly permutes the positions of every reactor on the
// field via atlas-reactors' POST .../reactors/shuffle.
func (p *ProcessorImpl) ShuffleReactors(f field.Model) error {
	_, err := requestShuffleReactors(p.ctx, f)(p.l, p.ctx)
	if err != nil {
		return fmt.Errorf("failed to shuffle reactors: %w", err)
	}
	return nil
}

// produceHitCommand produces a HIT command to COMMAND_TOPIC_REACTOR
func (p *ProcessorImpl) produceHitCommand(f field.Model, reactorId uint32, characterId uint32) error {
	key := producer.CreateKey(int(reactorId))
	value := &Command[HitCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      CommandTypeHit,
		Body: HitCommandBody{
			ReactorId:   reactorId,
			CharacterId: characterId,
		},
	}
	mp := producer.SingleMessageProvider(key, value)
	return produceToCommandTopic(p.l, p.ctx)(mp)
}

// produceToCommandTopic produces messages to the reactor command topic
func produceToCommandTopic(l logrus.FieldLogger, ctx context.Context) func(provider model.Provider[[]kafka.Message]) error {
	sd := producer.SpanHeaderDecorator(ctx)
	td := producer.TenantHeaderDecorator(ctx)
	ed := producer.EnvHeaderDecorator(ctx)
	return producer.Produce(l)(producer.ManagerWriterProvider(l)(EnvCommandTopic))(sd, td, ed)
}

// Command represents a command sent to atlas-reactors
type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// HitCommandBody represents the body of a HIT command
type HitCommandBody struct {
	ReactorId   uint32 `json:"reactorId"`
	CharacterId uint32 `json:"characterId"`
	Stance      uint16 `json:"stance"`
	SkillId     uint32 `json:"skillId"`
}
