package playernpc

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Processor resolves a character's current location for the
// deploy_player_npc default-map case (FR-6.2).
type Processor interface {
	GetCurrentLocation(characterId uint32) (world.Id, _map.Id, channel.Id, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new location processor.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetCurrentLocation returns the character's durable location as tracked by
// atlas-maps.
func (p *ProcessorImpl) GetCurrentLocation(characterId uint32) (world.Id, _map.Id, channel.Id, error) {
	rm, err := requestLocationByCharacterId(p.ctx, characterId)(p.l, p.ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	return rm.WorldId, rm.MapId, rm.ChannelId, nil
}
