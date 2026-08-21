package location

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Model is the state-bearing projection of atlas-maps's character location.
// field.Model has nowhere to carry the presence discriminator, so /find reads
// this instead; GetField is unchanged for its existing callers.
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

// NewModelForTest constructs a Model directly. Only call from a test;
// production code builds one through Get.
func NewModelForTest(characterId uint32, w world.Id, ch channel.Id, m _map.Id, instance uuid.UUID, state characterconst.PresenceState) Model {
	return Model{characterId: characterId, worldId: w, channelId: ch, mapId: m, instance: instance, state: state}
}
