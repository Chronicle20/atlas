package field

import (
	"atlas-maps/map/character"
	"context"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor exposes the business logic behind the field listing endpoint:
// filtering live field occupancy by the optional world/channel/map filters
// and returning it in deterministic order.
type Processor interface {
	// GetFields returns every live field occupancy belonging to t, constrained
	// by the optional worldId/channelId/mapId filters (nil means
	// unconstrained), sorted by world, channel, map, then instance-id string.
	GetFields(t tenant.Model, worldId *world.Id, channelId *channel.Id, mapId *_map.Id) []character.FieldOccupancy
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor constructs a Processor scoped to the supplied tenant context.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetFields loads every live field occupancy for t, drops entries that do not
// match a supplied filter, and sorts the survivors by world, channel, map,
// then instance-id string.
func (p *ProcessorImpl) GetFields(t tenant.Model, worldId *world.Id, channelId *channel.Id, mapId *_map.Id) []character.FieldOccupancy {
	occ := character.NewProcessor(p.l, p.ctx).GetFieldsWithCharacters(t)

	models := make([]character.FieldOccupancy, 0, len(occ))
	for _, o := range occ {
		if worldId != nil && o.Field.WorldId() != *worldId {
			continue
		}
		if channelId != nil && o.Field.ChannelId() != *channelId {
			continue
		}
		if mapId != nil && o.Field.MapId() != *mapId {
			continue
		}
		models = append(models, o)
	}

	sort.Slice(models, func(i, j int) bool {
		if models[i].Field.WorldId() != models[j].Field.WorldId() {
			return models[i].Field.WorldId() < models[j].Field.WorldId()
		}
		if models[i].Field.ChannelId() != models[j].Field.ChannelId() {
			return models[i].Field.ChannelId() < models[j].Field.ChannelId()
		}
		if models[i].Field.MapId() != models[j].Field.MapId() {
			return models[i].Field.MapId() < models[j].Field.MapId()
		}
		return models[i].Field.Instance().String() < models[j].Field.Instance().String()
	})

	return models
}
