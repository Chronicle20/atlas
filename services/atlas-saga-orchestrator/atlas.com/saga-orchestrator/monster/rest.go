package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"strconv"
)

// SpawnInputRestModel represents the input for spawning a monster.
type SpawnInputRestModel struct {
	Id        string `json:"-"`
	MonsterId uint32 `json:"monsterId"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
	Fh        int16  `json:"fh"`
	Team      int8   `json:"team"`
}

func (r SpawnInputRestModel) GetName() string {
	return "monsters"
}

func (r SpawnInputRestModel) GetID() string {
	return r.Id
}

// SpawnResponseRestModel represents the response from spawning a monster.
type SpawnResponseRestModel struct {
	Id                 string `json:"-"`
	UniqueId           uint32 `json:"uniqueId"`
	WorldId            byte   `json:"worldId"`
	ChannelId          byte   `json:"channelId"`
	MapId              uint32 `json:"mapId"`
	MonsterId          uint32 `json:"monsterId"`
	ControlCharacterId uint32 `json:"controlCharacterId"`
	X                  int16  `json:"x"`
	Y                  int16  `json:"y"`
	Fh                 int16  `json:"fh"`
	Stance             byte   `json:"stance"`
	Team               int8   `json:"team"`
	MaxHp              uint32 `json:"maxHp"`
	Hp                 uint32 `json:"hp"`
	MaxMp              uint32 `json:"maxMp"`
	Mp                 uint32 `json:"mp"`
}

func (r SpawnResponseRestModel) GetName() string {
	return "monsters"
}

func (r SpawnResponseRestModel) GetID() string {
	return r.Id
}

// SpawnRequest contains the parameters needed to spawn a monster.
type SpawnRequest struct {
	WorldId   world.Id
	ChannelId channel.Id
	MapId     uint32
	MonsterId uint32
	X         int16
	Y         int16
	Fh        int16
	Team      int8
}

func (r SpawnRequest) ToRestModel() SpawnInputRestModel {
	return SpawnInputRestModel{
		Id:        strconv.FormatUint(uint64(r.MonsterId), 10),
		MonsterId: r.MonsterId,
		X:         r.X,
		Y:         r.Y,
		Fh:        r.Fh,
		Team:      r.Team,
	}
}
