package rps

import (
	rpsMsg "atlas-channel/kafka/message/rps"
	producer2 "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// Processor emits COMMAND_TOPIC_RPS commands on behalf of the serverbound
// RPS_ACTION handler.
type Processor interface {
	Select(characterId uint32, worldId world.Id, channelId channel.Id, throw byte) error
	Continue(characterId uint32, worldId world.Id, channelId channel.Id) error
	Collect(characterId uint32, worldId world.Id, channelId channel.Id) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

// Select sends a SELECT command carrying the player's raw throw byte.
func (p *ProcessorImpl) Select(characterId uint32, worldId world.Id, channelId channel.Id, throw byte) error {
	p.l.Debugf("Sending RPS SELECT command for character [%d] throw [%d].", characterId, throw)
	return producer2.ProviderImpl(p.l)(p.ctx)(rpsMsg.EnvCommandTopic)(SelectCommandProvider(characterId, worldId, channelId, throw))
}

// Continue sends a CONTINUE command.
func (p *ProcessorImpl) Continue(characterId uint32, worldId world.Id, channelId channel.Id) error {
	p.l.Debugf("Sending RPS CONTINUE command for character [%d].", characterId)
	return producer2.ProviderImpl(p.l)(p.ctx)(rpsMsg.EnvCommandTopic)(ContinueCommandProvider(characterId, worldId, channelId))
}

// Collect sends a COLLECT command. atlas-rps treats this as collect-or-forfeit
// depending on session status - it is also the command the channel emits for
// the client's EXIT sub-op (there is no dedicated collect sub-op on the wire).
func (p *ProcessorImpl) Collect(characterId uint32, worldId world.Id, channelId channel.Id) error {
	p.l.Debugf("Sending RPS COLLECT command for character [%d].", characterId)
	return producer2.ProviderImpl(p.l)(p.ctx)(rpsMsg.EnvCommandTopic)(CollectCommandProvider(characterId, worldId, channelId))
}
