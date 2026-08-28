package location

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Builder struct{ m Model }

func NewBuilder(characterId uint32) *Builder {
	return &Builder{m: Model{characterId: characterId}}
}

func (b *Builder) SetWorldId(v world.Id) *Builder                   { b.m.worldId = v; return b }
func (b *Builder) SetChannelId(v channel.Id) *Builder               { b.m.channelId = v; return b }
func (b *Builder) SetMapId(v _map.Id) *Builder                      { b.m.mapId = v; return b }
func (b *Builder) SetInstance(v uuid.UUID) *Builder                 { b.m.instance = v; return b }
func (b *Builder) SetState(v characterconst.PresenceState) *Builder { b.m.state = v; return b }
func (b *Builder) SetField(f field.Model) *Builder {
	b.m.worldId = f.WorldId()
	b.m.channelId = f.ChannelId()
	b.m.mapId = f.MapId()
	b.m.instance = f.Instance()
	return b
}
func (b *Builder) Build() Model { return b.m }
