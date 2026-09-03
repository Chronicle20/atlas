package npc_spawn

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// SpawnInputRestModel represents the input for placing a scripted NPC on a field.
type SpawnInputRestModel struct {
	Id            string `json:"-"`
	NpcId         uint32 `json:"npcId"`
	X             int16  `json:"x"`
	Y             int16  `json:"y"`
	Fh            int16  `json:"fh"`
	SpawnIfAbsent bool   `json:"spawnIfAbsent,omitempty"`
}

func (r SpawnInputRestModel) GetName() string {
	return "npcs"
}

func (r SpawnInputRestModel) GetID() string {
	return r.Id
}

// SpawnResponseRestModel represents the response from placing an NPC on a field.
type SpawnResponseRestModel struct {
	Id        string     `json:"-"`
	UniqueId  uint32     `json:"uniqueId"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	NpcId     uint32     `json:"npcId"`
	X         int16      `json:"x"`
	Y         int16      `json:"y"`
	Fh        int16      `json:"fh"`
}

func (r SpawnResponseRestModel) GetName() string {
	return "npcs"
}

func (r SpawnResponseRestModel) GetID() string {
	return r.Id
}

func (r *SpawnResponseRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Required JSON:API relationship stubs (libs/atlas-rest gotcha): api2go errors
// out decoding any response unless the target implements these, even with no
// relationships present.
func (r *SpawnResponseRestModel) SetToOneReferenceID(_, _ string) error { return nil }

func (r *SpawnResponseRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// SpawnRequest contains the parameters needed to place an NPC on a field.
type SpawnRequest struct {
	WorldId       world.Id
	ChannelId     channel.Id
	MapId         _map.Id
	NpcId         uint32
	X             int16
	Y             int16
	Fh            int16
	SpawnIfAbsent bool
}

func (r SpawnRequest) ToRestModel() SpawnInputRestModel {
	return SpawnInputRestModel{
		Id:            strconv.FormatUint(uint64(r.NpcId), 10),
		NpcId:         r.NpcId,
		X:             r.X,
		Y:             r.Y,
		Fh:            r.Fh,
		SpawnIfAbsent: r.SpawnIfAbsent,
	}
}
