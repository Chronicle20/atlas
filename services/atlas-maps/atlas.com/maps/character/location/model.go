package location

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Model struct {
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	mapId       _map.Id
	instance    uuid.UUID
	state       characterconst.PresenceState
}

func (m Model) CharacterId() uint32                 { return m.characterId }
func (m Model) WorldId() world.Id                   { return m.worldId }
func (m Model) ChannelId() channel.Id               { return m.channelId }
func (m Model) MapId() _map.Id                      { return m.mapId }
func (m Model) Instance() uuid.UUID                 { return m.instance }
func (m Model) State() characterconst.PresenceState { return m.state }

func (m Model) Field() field.Model {
	return field.NewBuilder(m.worldId, m.channelId, m.mapId).SetInstance(m.instance).Build()
}

// ToEntity projects the immutable Model into its persistence entity for the
// given tenant. Callers (administrator) supply tenantId because the domain
// Model deliberately does not carry tenant identity.
func (m Model) ToEntity(tenantId uuid.UUID) entity {
	return entity{
		TenantId:    tenantId,
		CharacterId: m.characterId,
		WorldId:     m.worldId,
		ChannelId:   m.channelId,
		MapId:       m.mapId,
		Instance:    m.instance,
		State:       string(m.state),
		UpdatedAt:   time.Now(),
	}
}
