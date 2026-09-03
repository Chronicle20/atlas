package npc

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type ModelBuilder struct {
	m Model
}

func NewModelBuilder() *ModelBuilder { return &ModelBuilder{} }

func (b *ModelBuilder) SetUniqueId(id uint32) *ModelBuilder  { b.m.uniqueId = id; return b }
func (b *ModelBuilder) SetWorldId(id world.Id) *ModelBuilder { b.m.worldId = id; return b }
func (b *ModelBuilder) SetChannelId(id channel.Id) *ModelBuilder {
	b.m.channelId = id
	return b
}
func (b *ModelBuilder) SetMapId(id _map.Id) *ModelBuilder      { b.m.mapId = id; return b }
func (b *ModelBuilder) SetInstance(id uuid.UUID) *ModelBuilder { b.m.instance = id; return b }
func (b *ModelBuilder) SetField(f field.Model) *ModelBuilder {
	b.m.worldId = f.WorldId()
	b.m.channelId = f.ChannelId()
	b.m.mapId = f.MapId()
	b.m.instance = f.Instance()
	return b
}
func (b *ModelBuilder) SetNpcId(id uint32) *ModelBuilder { b.m.npcId = id; return b }
func (b *ModelBuilder) SetX(x int16) *ModelBuilder       { b.m.x = x; return b }
func (b *ModelBuilder) SetY(y int16) *ModelBuilder       { b.m.y = y; return b }
func (b *ModelBuilder) SetFh(fh int16) *ModelBuilder     { b.m.fh = fh; return b }

func (b *ModelBuilder) Build() Model { return b.m }
