package portal

import (
	"context"

	"atlas-saga-orchestrator/kafka/message"
	portalMsg "atlas-saga-orchestrator/kafka/message/portal"
	"atlas-saga-orchestrator/kafka/producer"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for portal blocking operations
type Processor interface {
	// BlockAndEmit sends a command to block a portal for a character
	BlockAndEmit(characterId uint32, mapId _map.Id, portalId uint32) error
	// Block adds a block command to the message buffer
	Block(mb *message.Buffer) func(characterId uint32, mapId _map.Id, portalId uint32) error
	// UnblockAndEmit sends a command to unblock a portal for a character
	UnblockAndEmit(characterId uint32, mapId _map.Id, portalId uint32) error
	// Unblock adds an unblock command to the message buffer
	Unblock(mb *message.Buffer) func(characterId uint32, mapId _map.Id, portalId uint32) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	p   producer.Provider
}

// NewProcessor creates a new portal processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   producer.ProviderImpl(l)(ctx),
	}
}

// BlockAndEmit sends a Kafka command to atlas-portals to block a portal for a character
func (p *ProcessorImpl) BlockAndEmit(characterId uint32, mapId _map.Id, portalId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Block(mb)(characterId, mapId, portalId)
	})
}

// Block adds a block command to the message buffer
func (p *ProcessorImpl) Block(mb *message.Buffer) func(characterId uint32, mapId _map.Id, portalId uint32) error {
	return func(characterId uint32, mapId _map.Id, portalId uint32) error {
		return mb.Put(portalMsg.EnvCommandTopic, BlockCommandProvider(characterId, mapId, portalId))
	}
}

// UnblockAndEmit sends a Kafka command to atlas-portals to unblock a portal for a character
func (p *ProcessorImpl) UnblockAndEmit(characterId uint32, mapId _map.Id, portalId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Unblock(mb)(characterId, mapId, portalId)
	})
}

// Unblock adds an unblock command to the message buffer
func (p *ProcessorImpl) Unblock(mb *message.Buffer) func(characterId uint32, mapId _map.Id, portalId uint32) error {
	return func(characterId uint32, mapId _map.Id, portalId uint32) error {
		return mb.Put(portalMsg.EnvCommandTopic, UnblockCommandProvider(characterId, mapId, portalId))
	}
}
